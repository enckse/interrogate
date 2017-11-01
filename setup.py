from setuptools import setup

setup(
    name='survey',
    version='__VERSION__',
    long_description="survey that is fully deployable without internet",
    packages=['survey'],
    include_package_data=True,
    zip_safe=False,
    install_requires=['Flask'],
    entry_points={
        'console_scripts': [
            'survey = survey.survey:main',
        ],
    },
)
