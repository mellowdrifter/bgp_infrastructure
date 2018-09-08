#!/usr/bin/env python3
import subprocess
import re


def getSubnets(family):
    regex = r'(?<=\/)\d{1,3}'
    if family == 4:
        routeDict = {'8': 0, '9': 0, '10': 0, '11': 0, '12': 0, '13': 0,
                     '14': 0, '15': 0, '16': 0, '17': 0, '18': 0, '19': 0,
                     '20': 0, '21': 0, '22': 0, '23': 0, '24': 0}
        routes = subprocess.check_output("/usr/sbin/birdc 'show route' | awk {'print $1'} | grep -v unreachable", shell=True).decode("utf-8")
    elif family == 6:
        routeDict = {'8': 0,  '9': 0,  '10': 0,  '11': 0,  '12': 0,  '13': 0,
                     '14': 0,  '15': 0,  '16': 0,  '17': 0,  '18': 0,  '19': 0,
                     '20': 0,  '21': 0,  '22': 0,  '23': 0,  '24': 0,  '25': 0,
                     '26': 0,  '27': 0,  '28': 0,  '29': 0,  '30': 0,  '31': 0,
                     '32': 0,  '33': 0,  '34': 0,  '35': 0,  '36': 0,  '37': 0,
                     '38': 0,  '39': 0,  '40': 0,  '41': 0,  '42': 0,  '43': 0,
                     '44': 0,  '45': 0,  '46': 0,  '47': 0,  '48': 0}
        routes = subprocess.check_output("/usr/sbin/birdc6 'show route' | awk {'print $1'} | grep -v unreachable", shell=True).decode("utf-8")
    else:
        return False
    routes = routes.rstrip()
    routeList = re.findall(regex, routes)
    for route in routeList:
        routeDict[route] += 1
    return routeDict

def getTotals(family):
    if family == 4:
        total = subprocess.check_output("/usr/sbin/birdc 'show route count' | grep 'routes' | awk {'print $3, $6'}", shell=True).decode("utf-8")
    elif family == 6:
        total = subprocess.check_output("/usr/sbin/birdc6 'show route count' | grep 'routes' | awk {'print $3, $6'}", shell=True).decode("utf-8")

    return total.split()

def getSrcAS():
    as4  = subprocess.check_output("/usr/sbin/birdc 'show route primary' | awk '{print $NF}' | tr -d '[]ASie?' ", shell=True).decode("utf-8")
    as6  = subprocess.check_output("/usr/sbin/birdc6 'show route primary' | awk '{print $NF}' | tr -d '[]ASie?' ", shell=True).decode("utf-8")
    as4  = set(as4.split())         # Total number of unique IPv4 source AS numbers
    as6  = set(as6.split())         # Total number of unique IPv6 source AS numbers
    as10 = as4.union(as6)           # Join two sets together for total unique source AS numbers
    as4_only = as4 - as6            # IPv4-only source AS
    as6_only = as6 - as4            # IPv6-only source AS
    as_both = as4.intersection(as6) # Source AS originating both IPv4 and IPv6
    return len(as4), len(as6), len(as10), len(as4_only), len(as6_only), len(as_both)

def getPeers(family):
    if family == 4:
        peers = subprocess.check_output("/usr/sbin/birdc 'show protocols' | awk {'print $1'} | grep -Ev 'BIRD|device1|name|info'", shell=True).decode("utf-8")
        state = subprocess.check_output("/usr/sbin/birdc 'show protocols' | awk {'print $6'} | grep -Ev 'BIRD|device1|name|info'", shell=True).decode("utf-8")
    elif family == 6:
        peers = subprocess.check_output("/usr/sbin/birdc6 'show protocols' | awk {'print $1'} | grep -Ev 'BIRD|device1|name|info'", shell=True).decode("utf-8")
        state = subprocess.check_output("/usr/sbin/birdc6 'show protocols' | awk {'print $6'} | grep -Ev 'BIRD|device1|name|info'", shell=True).decode("utf-8")
    peers = peers.rstrip()
    state = state.rstrip()
    peers = peers.splitlines()
    state = state.splitlines()
    state = filter(None, state)
    peerState = {}
    for p,s in zip(peers, state):
        peerState[p] = s
    peerState['peersConfigured'] = len(peers)
    peers_up = 0
    for status in state:
        if 'Established' in status:
            peers_up += 1
    peerState['peersUp'] = peers_up

    return peerState

def getMem(family):
    values = {}
    if family == 4:
        mem = subprocess.check_output("/usr/sbin/birdc 'show mem'", shell=True).decode("utf-8")
    elif family == 6:
        mem = subprocess.check_output("/usr/sbin/birdc6 'show mem'", shell=True).decode("utf-8")
    else:
        return False
    mem = mem.splitlines()
    routes = re.match(r'^(Routing tables):\s{1,50}(\d{1,5}\s{1,10}\w{1,10}$)', mem[2])
    att = re.match(r'^(Route attributes):\s{1,50}(\d{1,5}\s{1,10}\w{1,10}$)', mem[3])
    roa = re.match(r'^(ROA tables):\s{1,50}(\d{1,5}\s{1,10}\w{1,10}$)', mem[4])
    protocols = re.match(r'^(Protocols):\s{1,50}(\d{1,5}\s{1,10}\w{1,10}$)', mem[5])
    total = re.match(r'^(Total):\s{1,50}(\d{1,5}\s{1,10}\w{1,10})', mem[6])
    values[routes.group(1)] = routes.group(2).replace(" ", "")
    values[att.group(1)] = att.group(2).replace(" ", "")
    values[roa.group(1)] = roa.group(2).replace(" ", "")
    values[protocols.group(1)] = protocols.group(2).replace(" ", "")

    values[total.group(1)] = total.group(2).replace(" ", "")
    return values


if __name__ == "__main__":
    print('IPv6 Subnets\n============\n', getSubnets(6))
    print('IPv4 Subnets\n============\n', getSubnets(4))
    print('IPv6 total count\n=========\n', getTotals(6))
    print('IPv4 total count\n=========\n', getTotals(4))
    print('None test\n=========\n', getSubnets('r'))
