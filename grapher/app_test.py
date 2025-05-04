from unittest import TestCase, mock
import logging
import app

class TotalTime:
    def __init__(self, time, v4_values, v6_values):
        self.time = time
        self.v4_values = v4_values
        self.v6_values = v6_values

class LineGraphRequest:
    def __init__(self):
        self.totals_time = []


class TestFilterOutliers(TestCase):
    def setUp(self):
        # Mock WINDOW and THRESHOLD (adjust values as needed)
        self.WINDOW = 2
        self.THRESHOLD = 10  # 10% threshold
        self.original_WINDOW = app.WINDOW
        self.original_THRESHOLD = app.THRESHOLD
        app.WINDOW = self.WINDOW
        app.THRESHOLD = self.THRESHOLD

    def tearDown(self):
        # Reset original values
        app.WINDOW = self.original_WINDOW
        app.THRESHOLD = self.original_THRESHOLD

    def test_middle_outlier_replaced(self):
        data = [
            TotalTime(1, 100, 200),
            TotalTime(2, 100, 200),
            TotalTime(3, 121, 200),  # IPv4 outlier (21% > 10%)
            TotalTime(4, 100, 200),
            TotalTime(5, 100, 200),
        ]
        result = app.filter_outliers(data)
        self.assertEqual(result.totals_time[2].v4_values, 0)  # Replaced
        self.assertEqual(result.totals_time[2].v6_values, 200)  # Unchanged

    def test_edge_element_handling(self):
        data = [
            TotalTime(1, 200, 200),  # First element (neighbors: [100, 100])
            TotalTime(2, 100, 200),
            TotalTime(3, 100, 200),
        ]
        app.WINDOW = 1  # Test with smaller window
        result = app.filter_outliers(data)
        self.assertEqual(result.totals_time[0].v4_values, 0)  # (200-100)/100 = 100% > 10%

    def test_zero_median_handling(self):
        data = [
            TotalTime(1, 0, 0),
            TotalTime(2, 0, 0),
            TotalTime(3, 5, 0),  # (5-0)/1 = 500% > 10%
            TotalTime(4, 0, 0),
            TotalTime(5, 0, 0),
        ]
        result = app.filter_outliers(data)
        self.assertEqual(result.totals_time[2].v4_values, 0)

    @mock.patch("app.logging")
    def test_logging_on_outlier(self, mock_logging):
        data = [TotalTime(1, 200, 0), TotalTime(2, 100, 0)]
        app.WINDOW = 1
        app.filter_outliers(data)
        mock_logging.info.assert_called_with("Replacing IPv4 outlier 200 with 0")

    def test_empty_neighbors_retains_value(self):
        data = [TotalTime(1, 100, 200)]  # No neighbors
        result = app.filter_outliers(data)
        self.assertEqual(result.totals_time[0].v4_values, 100)
