#!/usr/bin/env python

# -*- coding: utf-8 -*-
from setuptools import setup, find_packages

setup(
    name="Music GRPC Events",
    version="1.0",
    packages=find_packages(),
    entry_points={
        'console_scripts': [
            'music-grpc-events = musicevents:main',
        ],
    },
)
