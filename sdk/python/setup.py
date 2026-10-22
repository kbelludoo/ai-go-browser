from setuptools import setup, find_packages

setup(
    name="aigo-browser",
    version="2.0.0",
    description="Ultra-lightweight native semantic browser (<35MB RAM) for AI agents and LLM pipelines",
    author="kbelludoo",
    packages=find_packages(),
    python_requires=">=3.8",
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
        "Topic :: Scientific/Engineering :: Artificial Intelligence",
        "Topic :: Internet :: WWW/HTTP :: Browsers",
    ],
)
