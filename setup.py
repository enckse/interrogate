#!/usr/bin/python
"""Setup script for survey tooling."""
from setuptools import setup, find_packages
import os

__pkg_name__ = "survey"
with open(os.path.join(__pkg_name__, "version.py")) as v_file:
    exec(v_file.read())

setup(
    name=__pkg_name__,
    version=__version__,
    long_description="survey that is fully deployable without internet",
    packages=[__pkg_name__],
    include_package_data=True,
    zip_safe=False,
    install_requires=['Flask'],
    entry_points={
        'console_scripts': [
            'survey = survey.survey:main',
        ],
    },
)
