#!/usr/bin/env python3

from typing import List, Text
import os
from concurrent import futures
import grapher_pb2 as pb
import grapher_pb2_grpc
import datetime
import grpc
import io
import logging
import sys
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
import matplotlib.ticker as ticker

_PORT = os.environ.get("PORT", "50051")

# Neighbour value threshold in percent
THRESHOLD = 2
v4THRESHOLD = 2
v6THRESHOLD = 5
# How many neighbours to consider
WINDOW = 20

# Modern Design Tokens (Dark Mode Default)
DARK_BG = "#0B0F19"
DARK_TEXT_MAIN = "#F8FAFC"
DARK_TEXT_MUTED = "#94A3B8"
DARK_TEXT_DIM = "#64748B"
DARK_GRID = "#1E293B"

LIGHT_BG = "#FFFFFF"
LIGHT_TEXT_MAIN = "#0F172A"
LIGHT_TEXT_MUTED = "#475569"
LIGHT_TEXT_DIM = "#94A3B8"
LIGHT_GRID = "#E2E8F0"

# Modern Vibrant Color Mappings for Legacy Names
LEGACY_COLOR_MAP = {
    "burlywood": "#F59E0B",   # Warm Amber
    "lightgreen": "#10B981",  # Vibrant Emerald
    "lightskyblue": "#06B6D4",# Clean Cyan
    "lightcoral": "#F43F5E",  # Modern Rose / Red
    "gold": "#6366F1",        # Modern Indigo
    "violet": "#8B5CF6",      # Modern Purple
    "linen": "#64748B",       # Modern Slate Grey
    "#238341": "#10B981",     # Legacy dark green -> Emerald
    "#0041A0": "#38BDF8",     # Legacy dark blue -> Electric Sky
}

# RPKI Color Palette
RPKI_COLORS = {
    "valid": "#10B981",       # Emerald green
    "unknown": "#64748B",     # Slate grey
    "invalid": "#F43F5E",     # Rose / Red
}

def map_color(col: str) -> str:
    """Map legacy HTML / dark matplotlib colors to modern high-contrast tokens."""
    return LEGACY_COLOR_MAP.get(col.lower() if isinstance(col, str) else col, col)


class Grapher(grapher_pb2_grpc.GrapherServicer):
    """Provides methods that implement functionality of the grapher server."""

    def GetLineGraph(self, request, context):
        logging.info("Request received for GetLineGraph")
        return get_line_graph(request)

    def GetPieChart(self, request, context):
        logging.info("Request received for GetPieChart")
        return get_pie_chart(request)

    def GetRPKI(self, request, context):
        logging.info("Request received for GetRPKI")
        return get_rpki(request)

    def TestRPC(self, request, context):
        logging.info("Received request: %s", request)
        return pb.TestResponse(testresponse="something")


# Remove values outside of the threshold. When restarting BGP collectors it takes a few minutes to get full total and makes horrible graphs.
def filter_outliers(data: List[pb.TotalTime]) -> pb.LineGraphRequest:
    mod = sys.modules[__name__]
    win = getattr(mod, 'WINDOW', WINDOW)
    v4_thresh = getattr(mod, 'THRESHOLD', v4THRESHOLD)
    v6_thresh = getattr(mod, 'v6THRESHOLD', v6THRESHOLD)

    v4 = []
    v6 = []
    n = len(data)
    for i in range(n):
        v4.append(data[i].v4_values)
        v6.append(data[i].v6_values)

    filtered_v4 = []
    filtered_v6 = []

    for i in range(n):
        start = max(0, i - win)
        end = min(n, i + win + 1)

        neighbors_v4 = v4[start:i] + v4[i+1:end]
        neighbors_v6 = v6[start:i] + v6[i+1:end]

        if not neighbors_v4 or not neighbors_v6:
            filtered_v4.append(v4[i])
            filtered_v6.append(v6[i])
            continue

        median_v4 = sorted(neighbors_v4)[len(neighbors_v4) // 2]
        median_v6 = sorted(neighbors_v6)[len(neighbors_v6) // 2]

        is_outlier_v4 = abs(v4[i] - median_v4) / (median_v4 or 1) > (v4_thresh / 100)
        is_outlier_v6 = abs(v6[i] - median_v6) / (median_v6 or 1) > (v6_thresh / 100)

        if is_outlier_v4:
            logging.info(f"Replacing IPv4 outlier {v4[i]} with 0")
            filtered_v4.append(0)
        else:
            filtered_v4.append(v4[i])

        if is_outlier_v6:
            logging.info(f"Replacing IPv6 outlier {v6[i]} with 0")
            filtered_v6.append(0)
        else:
            filtered_v6.append(v6[i])

    totals = pb.LineGraphRequest()
    for i in range(n):
        msg = pb.TotalTime()
        msg.time = data[i].time
        msg.v4_values = filtered_v4[i]
        msg.v6_values = filtered_v6[i]
        totals.totals_time.extend([msg])

    logging.info("Filtered outliers")
    return totals


def setup_theme(theme: str):
    if theme == "light":
        return LIGHT_BG, LIGHT_TEXT_MAIN, LIGHT_TEXT_MUTED, LIGHT_TEXT_DIM, LIGHT_GRID
    return DARK_BG, DARK_TEXT_MAIN, DARK_TEXT_MUTED, DARK_TEXT_DIM, DARK_GRID


def add_header_footer(fig, title: str, subtitle: str, text_main: str, text_muted: str, text_dim: str):
    # Main Title (Top Left)
    fig.text(0.08, 0.935, title, fontsize=17.5, fontweight='bold',
             color=text_main, ha='left', va='top', fontfamily='sans-serif')
    # Subtitle / Metric summary (Left)
    if subtitle:
        fig.text(0.08, 0.88, subtitle, fontsize=12, fontweight='normal',
                 color=text_muted, ha='left', va='top', fontfamily='sans-serif')

    # Prominent Author & Sites (Top Right - High Contrast, Large and Readable)
    fig.text(0.92, 0.935, "Darren O'Connor", fontsize=22, fontweight='bold',
             color=text_main, ha='right', va='top', fontfamily='sans-serif')
    fig.text(0.92, 0.88, "daz.bgpstuff.net  •  mellowd.dev", fontsize=14.5, fontweight='bold',
             color="#38BDF8", ha='right', va='top', fontfamily='sans-serif')

    # Footer Left (Clean telemetry notice, no duplicate author/domain data)
    fig.text(0.08, 0.04, "bgpstuff.net  •  Global BGP Routing Telemetry", fontsize=12,
             color=text_dim, ha='left', va='bottom', fontfamily='sans-serif')


def get_line_graph(request: pb.LineGraphRequest) -> pb.GrapherResponse:
    logging.info('running get_line_graph')

    v4Dates = []
    v6Dates = []
    dates = []
    v4totals = []
    v6totals = []
    prefixes = []
    graphs = pb.GrapherResponse()

    # Filter outliers
    totals = filter_outliers(request.totals_time).totals_time
    for i in range(len(totals)):
        if totals[i].v4_values != 0:
            v4Dates.append(datetime.datetime.fromtimestamp(totals[i].time))
            v4totals.append(totals[i].v4_values)
        if totals[i].v6_values != 0:
            v6Dates.append(datetime.datetime.fromtimestamp(totals[i].time))
            v6totals.append(totals[i].v6_values)
            
    prefixes.append(v4totals)
    prefixes.append(v6totals)
    dates.append(v4Dates)
    dates.append(v6Dates)

    j = 0
    for metadata in request.metadatas:
        title = metadata.title
        x = metadata.x_axis or 13.33
        y = metadata.y_axis or 7.5
        # Normalize legacy 12x10 to 16:9 for modern social card display
        if x == 12 and y == 10:
            x, y = 13.33, 7.5
        colour = map_color(metadata.colour) if metadata.colour else ("#10B981" if j == 0 else "#38BDF8")
        theme = metadata.theme

        bg_col, text_main, text_muted, text_dim, grid_col = setup_theme(theme)

        current_dates = dates[j] if j < len(dates) else []
        current_prefixes = prefixes[j] if j < len(prefixes) else []

        fig = plt.figure(figsize=(x, y), dpi=200, facecolor=bg_col)
        
        # Calculate summary metrics if data exists
        subtitle = ""
        if current_prefixes:
            start_val = current_prefixes[0]
            end_val = current_prefixes[-1]
            min_val = min(current_prefixes)
            max_val = max(current_prefixes)
            delta = end_val - start_val
            pct = (delta / start_val * 100) if start_val else 0
            sign = "+" if delta >= 0 else ""
            subtitle = f"Latest: {end_val:,} prefixes  •  Movement: {sign}{delta:,} ({sign}{pct:.2f}%)"

        # Header and Footer with prominent author branding
        add_header_footer(fig, title, subtitle, text_main, text_muted, text_dim)

        if current_dates and current_prefixes:
            ax = fig.add_axes([0.11, 0.12, 0.81, 0.72], facecolor=bg_col)
            
            # Format X axis dynamically based on timespan
            days_span = (current_dates[-1] - current_dates[0]).days if len(current_dates) > 1 else 1
            if days_span <= 7:
                ax.xaxis.set_major_locator(mdates.DayLocator(interval=1))
                ax.xaxis.set_major_formatter(mdates.DateFormatter('%d %b'))
            elif days_span <= 35:
                ax.xaxis.set_major_locator(mdates.DayLocator(interval=5))
                ax.xaxis.set_major_formatter(mdates.DateFormatter('%d %b'))
            elif days_span <= 180:
                ax.xaxis.set_major_locator(mdates.MonthLocator(interval=1))
                ax.xaxis.set_major_formatter(mdates.DateFormatter('%b %Y'))
            else:
                ax.xaxis.set_major_locator(mdates.MonthLocator(interval=2))
                ax.xaxis.set_major_formatter(mdates.DateFormatter('%b %Y'))

            # Format Y axis with comma separator
            ax.yaxis.set_major_formatter(ticker.FuncFormatter(lambda val, pos: f"{int(val):,}"))

            # Clean minimalist gridlines
            ax.grid(True, linestyle='--', linewidth=0.7, color=grid_col, alpha=0.9, axis='y')
            ax.grid(False, axis='x')
            for spine in ax.spines.values():
                spine.set_visible(False)

            ax.tick_params(axis='both', which='both', length=0)
            ax.tick_params(axis='x', labelsize=11, labelcolor=text_muted, pad=8)
            ax.tick_params(axis='y', labelsize=11, labelcolor=text_muted, pad=8)

            # Smooth curve + gradient area fill
            ax.plot(current_dates, current_prefixes, color=colour, linewidth=2.8, zorder=4)
            min_y = min(current_prefixes)
            max_y = max(current_prefixes)
            y_margin = (max_y - min_y) * 0.25 if max_y != min_y else 1000
            ax.set_ylim(min_y - y_margin * 0.6, max_y + y_margin)
            ax.fill_between(current_dates, current_prefixes, min_y - y_margin * 0.6,
                            color=colour, alpha=0.12, zorder=2)

            # Milestone marker on end point
            ax.scatter([current_dates[-1]], [current_prefixes[-1]], color=colour, s=70, zorder=5,
                       edgecolors=bg_col, linewidth=2)
            ax.annotate(f"  {current_prefixes[-1]:,}",
                        xy=(current_dates[-1], current_prefixes[-1]),
                        xytext=(0, 0), textcoords="offset points",
                        color=text_main, fontsize=12, fontweight='bold', va='center')

            # Start point dot
            ax.scatter([current_dates[0]], [current_prefixes[0]], color=colour, alpha=0.6, s=40, zorder=5)

        image = io.BytesIO()
        plt.savefig(image, format='png', facecolor=fig.get_facecolor(), edgecolor='none', dpi=200)
        image.seek(0)
        graph = graphs.images.add()
        graph.image = image.read()
        graph.title = title
        plt.close(fig)
        j += 1

    logging.info("Returning line graphs")
    return graphs


def get_pie_chart(request: pb.PieChartRequest) -> pb.GrapherResponse:
    logging.info('running get_pie_chart')

    pieCharts = pb.GrapherResponse()
    subnets = [list(request.subnets.v4_values), list(request.subnets.v6_values)]

    j = 0
    for metadata in request.metadatas:
        title = metadata.title
        x = metadata.x_axis or 13.33
        y = metadata.y_axis or 7.5
        # Normalize legacy 12x10 to 16:9 for modern social card display
        if x == 12 and y == 10:
            x, y = 13.33, 7.5
        labels = list(metadata.labels)
        raw_colours = list(metadata.colours)
        colours = [map_color(c) for c in raw_colours]
        theme = metadata.theme

        bg_col, text_main, text_muted, text_dim, grid_col = setup_theme(theme)
        current_subnets = subnets[j] if j < len(subnets) else []
        total = sum(current_subnets)

        fig = plt.figure(figsize=(x, y), dpi=200, facecolor=bg_col)
        
        # Header and Footer with prominent author branding
        add_header_footer(fig, title, f"Total Evaluated: {total:,} prefixes", text_main, text_muted, text_dim)

        if current_subnets and total > 0:
            # Left: Donut Chart
            ax_donut = fig.add_axes([0.06, 0.10, 0.42, 0.74], facecolor=bg_col)
            ax_donut.pie(
                current_subnets,
                colors=colours,
                startangle=140,
                wedgeprops=dict(width=0.34, edgecolor=bg_col, linewidth=3.5),
            )
            # Center summary text
            ax_donut.text(0, 0.08, f"{total:,}", ha='center', va='center',
                          fontsize=20, fontweight='bold', color=text_main, fontfamily='sans-serif')
            ax_donut.text(0, -0.09, "TOTAL PREFIXES", ha='center', va='center',
                          fontsize=10, fontweight='bold', color=text_dim, fontfamily='sans-serif')

            # Right: Clean Structured Legend Card with wide columns
            ax_legend = fig.add_axes([0.50, 0.12, 0.42, 0.69], facecolor=bg_col)
            ax_legend.set_xlim(0, 1)
            ax_legend.set_ylim(0, 1)
            ax_legend.axis('off')

            n = len(current_subnets)
            y_step = 0.88 / max(n, 1)

            for i, (val, label, col) in enumerate(zip(current_subnets, labels, colours)):
                y_pos = 0.92 - (i * y_step)
                pct = (val / total * 100) if total else 0

                # Color pill
                ax_legend.add_patch(plt.Circle((0.04, y_pos), 0.022, color=col, transform=ax_legend.transAxes))

                # Label
                ax_legend.text(0.10, y_pos, label, transform=ax_legend.transAxes,
                               fontsize=12.5, fontweight='bold', color=text_main, va='center')

                # Value count
                ax_legend.text(0.74, y_pos, f"{val:,}", transform=ax_legend.transAxes,
                               fontsize=12, color=text_muted, ha='right', va='center')

                # Percentage badge
                ax_legend.text(0.98, y_pos, f"{pct:5.1f}%", transform=ax_legend.transAxes,
                               fontsize=13, fontweight='bold', color=text_main, ha='right', va='center')

                # Row divider line
                if i < n - 1:
                    ax_legend.plot([0.02, 0.98], [y_pos - (y_step * 0.48), y_pos - (y_step * 0.48)],
                                   color=grid_col, linewidth=0.8, transform=ax_legend.transAxes)

        image = io.BytesIO()
        plt.savefig(image, format='png', facecolor=fig.get_facecolor(), edgecolor='none', dpi=200)
        image.seek(0)
        pie = pieCharts.images.add()
        pie.image = image.read()
        pie.title = title
        plt.close(fig)
        j += 1

    logging.info("Returning pie charts")
    return pieCharts


def get_rpki(request: pb.RPKIRequest) -> pb.GrapherResponse:
    logging.info('running get_rpki')

    v4_rpki = [request.rpkis.v4_valid, request.rpkis.v4_unknown, request.rpkis.v4_invalid]
    v6_rpki = [request.rpkis.v6_valid, request.rpkis.v6_unknown, request.rpkis.v6_invalid]
    rpkis = [v4_rpki, v6_rpki]
    RPKICharts = pb.GrapherResponse()

    labels = ['Valid (ROA Match)', 'No ROA (Unknown)', 'Invalid (Mismatch)']
    colours = [RPKI_COLORS["valid"], RPKI_COLORS["unknown"], RPKI_COLORS["invalid"]]

    j = 0
    for metadata in request.metadatas:
        title = metadata.title
        x = metadata.x_axis or 13.33
        y = metadata.y_axis or 7.5
        # Normalize legacy 12x10 to 16:9 for modern social card display
        if x == 12 and y == 10:
            x, y = 13.33, 7.5
        theme = metadata.theme

        bg_col, text_main, text_muted, text_dim, grid_col = setup_theme(theme)
        current_rpki = rpkis[j] if j < len(rpkis) else []
        total = sum(current_rpki)

        fig = plt.figure(figsize=(x, y), dpi=200, facecolor=bg_col)
        
        # Header and Footer with prominent author branding
        add_header_footer(fig, title, f"Total Evaluated: {total:,} prefixes", text_main, text_muted, text_dim)

        if current_rpki and total > 0:
            # Left: Donut Chart
            ax_donut = fig.add_axes([0.06, 0.10, 0.42, 0.74], facecolor=bg_col)
            ax_donut.pie(
                current_rpki,
                colors=colours,
                startangle=140,
                wedgeprops=dict(width=0.34, edgecolor=bg_col, linewidth=3.5),
            )
            # Center summary text
            ax_donut.text(0, 0.08, f"{total:,}", ha='center', va='center',
                          fontsize=20, fontweight='bold', color=text_main, fontfamily='sans-serif')
            ax_donut.text(0, -0.09, "TOTAL PREFIXES", ha='center', va='center',
                          fontsize=10, fontweight='bold', color=text_dim, fontfamily='sans-serif')

            # Right: Clean Structured Legend Card with wide columns
            ax_legend = fig.add_axes([0.50, 0.14, 0.42, 0.65], facecolor=bg_col)
            ax_legend.set_xlim(0, 1)
            ax_legend.set_ylim(0, 1)
            ax_legend.axis('off')

            n = len(current_rpki)
            y_step = 0.85 / max(n, 1)

            for i, (val, label, col) in enumerate(zip(current_rpki, labels, colours)):
                y_pos = 0.85 - (i * y_step)
                pct = (val / total * 100) if total else 0

                # Color pill
                ax_legend.add_patch(plt.Circle((0.04, y_pos), 0.025, color=col, transform=ax_legend.transAxes))

                # Label
                ax_legend.text(0.10, y_pos, label, transform=ax_legend.transAxes,
                               fontsize=13.5, fontweight='bold', color=text_main, va='center')

                # Value count
                ax_legend.text(0.74, y_pos, f"{val:,}", transform=ax_legend.transAxes,
                               fontsize=13, color=text_muted, ha='right', va='center')

                # Percentage badge
                ax_legend.text(0.98, y_pos, f"{pct:5.1f}%", transform=ax_legend.transAxes,
                               fontsize=14, fontweight='bold', color=text_main, ha='right', va='center')

                # Row divider line
                if i < n - 1:
                    ax_legend.plot([0.02, 0.98], [y_pos - (y_step * 0.48), y_pos - (y_step * 0.48)],
                                   color=grid_col, linewidth=1.2, transform=ax_legend.transAxes)

        image = io.BytesIO()
        plt.savefig(image, format='png', facecolor=fig.get_facecolor(), edgecolor='none', dpi=200)
        image.seek(0)
        rpki = RPKICharts.images.add()
        rpki.image = image.read()
        rpki.title = title
        plt.close(fig)
        j += 1

    logging.info("Returning rpki charts")
    return RPKICharts


def _serve(port: Text):
    bind_address = f"[::]:{port}"
    server = grpc.server(futures.ThreadPoolExecutor())
    grapher_pb2_grpc.add_GrapherServicer_to_server(Grapher(), server)
    server.add_insecure_port(bind_address)
    server.start()
    logging.info("Listening on %s.", bind_address)
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    _serve(_PORT)
