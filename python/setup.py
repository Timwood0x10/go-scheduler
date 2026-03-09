from setuptools import setup, find_packages

setup(
    name="gpu-scheduler",
    version="0.0.1",
    description="GPU Scheduler Python SDK",
    author="GPU Scheduler Team",
    packages=find_packages(),
    install_requires=[
        "grpcio>=1.78.0",
        "protobuf>=4.21.0",
    ],
    python_requires=">=3.8",
)
