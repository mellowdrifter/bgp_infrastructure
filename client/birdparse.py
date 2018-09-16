#!/usr/bin/env python3
import subprocess
import re


def getSubnets():
    regex = r'(?<=\/)\d{1,3}'
    subnets4 = {}
    subnets6 = {}
    routes4 = subprocess.check_output("/usr/sbin/birdc 'show route' | awk {'print $1'} | grep -v unreachable", shell=True).decode("utf-8")
    routes6 = subprocess.check_output("/usr/sbin/birdc6 'show route' | awk {'print $1'} | grep -v unreachable", shell=True).decode("utf-8")
    for i in range(8, 25):
        subnets4[str(i)] = 0
    for i in range(8, 49):
        subnets6[str(i)] = 0
    routes4 = routes4.rstrip()
    routes6 = routes6.rstrip()
    routeList4 = re.findall(regex, routes4)
    routeList6 = re.findall(regex, routes6)
    for route in routeList4:
        subnets4[route] += 1
    for route in routeList6:
        subnets6[route] += 1

    subnet4 = []
    subnet6 = []
    for i in range(8, 25):
        subnet4.append(subnets4.get(str(i)))
    for i in range(8, 49):
        subnet6.append(subnets6.get(str(i)))

    return subnet4, subnet6

def getTotals():
    total4 = subprocess.check_output("/usr/sbin/birdc 'show route count' | grep 'routes' | awk {'print $3, $6'}", shell=True).decode("utf-8")
    total6 = subprocess.check_output("/usr/sbin/birdc6 'show route count' | grep 'routes' | awk {'print $3, $6'}", shell=True).decode("utf-8")

    return total4.split(), total6.split()

def getSrcAS():
    as4  = subprocess.check_output("/usr/sbin/birdc 'show route primary' | awk '{print $NF}' | tr -d '[]ASie?' | sed -n '1!p'", shell=True).decode("utf-8")
    as6  = subprocess.check_output("/usr/sbin/birdc6 'show route primary' | awk '{print $NF}' | tr -d '[]ASie?' | sed -n '1!p'", shell=True).decode("utf-8")
    as4  = set(as4.split())         # Total number of unique IPv4 source AS numbers
    as6  = set(as6.split())         # Total number of unique IPv6 source AS numbers
    as10 = as4.union(as6)           # Join two sets together for total unique source AS numbers
    as4_only = as4 - as6            # IPv4-only source AS
    as6_only = as6 - as4            # IPv6-only source AS
    as_both = as4.intersection(as6) # Source AS originating both IPv4 and IPv6

    return len(as4), len(as6), len(as10), len(as4_only), len(as6_only), len(as_both)

def getPeers(family):
    if family == 4:
        peers = int(subprocess.check_output("/usr/sbin/birdc 'show protocols' | awk {'print $1'} | grep -Ev 'BIRD|device1|name|info|kernel1' | wc -l", shell=True).decode("utf-8"))
        state = int(subprocess.check_output("/usr/sbin/birdc 'show protocols' | awk {'print $6'} | grep Established | wc -l", shell=True).decode("utf-8"))
    elif family == 6:
        peers = int(subprocess.check_output("/usr/sbin/birdc6 'show protocols' | awk {'print $1'} | grep -Ev 'BIRD|device1|name|info|kernel1' | wc -l", shell=True).decode("utf-8"))
        state = int(subprocess.check_output("/usr/sbin/birdc6 'show protocols' | awk {'print $6'} | grep Established | wc -l", shell=True).decode("utf-8"))

    return peers, state

def getLargeCommunitys():
    large4 = int(subprocess.check_output("/usr/sbin/birdc 'show route where bgp_large_community ~ [(*,*,*)]' | sed -n '1!p' | wc -l", shell=True).decode("utf-8"))
    large6 = int(subprocess.check_output("/usr/sbin/birdc6 'show route where bgp_large_community ~ [(*,*,*)]'| sed -n '1!p' | wc -l", shell=True).decode("utf-8"))

    return large4, large6

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
    print('Subnets\n============\n', getSubnets())
    print('Total count\n=========\n', getTotals())
    print('Source AS\n', getSrcAS())
    print('Peers\n', getPeers(4))
    print('Peers\n', getPeers(6))
    print('Large Comm\n', getLargeCommunitys())
    print('Peers\n', getMem(4))
    print('Peers\n', getMem(6))
