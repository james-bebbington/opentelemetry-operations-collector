#!/bin/bash

# The use of >/dev/null is to supress stdout, but allow stderr
# This is because installing the tools and building makes a lot of output

echo "Installing tools needed"

# Install git and maven for building the exporters
sudo apt-get update
sudo apt-get install -y git
sudo apt-get install -y maven

mkdir -p dist/prometheus_exporters

echo "Building Prometheus exporters"

# Clone mysqld exporter, build the binary and put it in the folder
echo "Cloning and building the MySQL exporter"

git clone https://github.com/prometheus/mysqld_exporter.git -q
make -C mysqld_exporter >/dev/null
cp mysqld_exporter/mysqld_exporter dist/prometheus_exporters/mysqld_exporter
rm -rf mysqld_exporter

# Clone Apache exporter, build the binary and put it in the folder
echo "Cloning and building the Apache exporter"

git clone https://github.com/Lusitaniae/apache_exporter.git -q
make -C apache_exporter >/dev/null
cp apache_exporter/apache_exporter dist/prometheus_exporters/apache_exporter
rm -rf apache_exporter

# Clone JVM exporter, build the binary and put it in the folder
echo "Cloning and building the JVM exporter"

git clone https://github.com/prometheus/jmx_exporter.git -q
# version is needed to figure out the name of the jar file generated by maven
version=$(sed -n -e 's#.*<version>\(.*-SNAPSHOT\)</version>#\1#p' jmx_exporter/pom.xml)
mvn -f jmx_exporter/pom.xml package >/dev/null
cp jmx_exporter/jmx_prometheus_httpserver/target/jmx_prometheus_httpserver-${version}-jar-with-dependencies.jar dist/prometheus_exporters/jmx_exporter.jar
rm -rf jmx_exporter

# StatsD exporter
echo "Cloning and building the StatsD exporter"

git clone https://github.com/prometheus/statsd_exporter.git -q
make -C statsd_exporter/ >/dev/null
cp statsd_exporter/statsd_exporter dist/prometheus_exporters/statsd_exporter
rm -rf statsd_exporter

echo "Prometheus exporters built"
