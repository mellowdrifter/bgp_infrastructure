#!/usr/bin/env python3

import psutil

def isRun(process: str):
    for proc in psutil.process_iter():
        if proc.name() == process:
            return True
    return False


if __name__ == '__main__':
    if isRun('bird'):
        print('bird is running')
    else:
        print('bird is not running!')

    if isRun('bird6'):
        print('bird6 is running')
    else:
        print('bird6 is not running!')
