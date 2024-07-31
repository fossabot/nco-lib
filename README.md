# Shared lib for other nco-* tools #

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ncotds_nco-lib&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=ncotds_nco-lib)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=ncotds_nco-lib&metric=coverage)](https://sonarcloud.io/summary/new_code?id=ncotds_nco-lib)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=ncotds_nco-lib&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=ncotds_nco-lib)
[![Build](https://github.com/ncotds/nco-lib/actions/workflows/build-release-assets.yml/badge.svg)](https://github.com/ncotds/nco-lib/actions/workflows/build-release-assets.yml)

> *"Netcool OMNIbus Object Server" - component of IBM Netcool stack, in-memory database to store alerts data*

... TBD

## Versioning

We use [SemVer](http://semver.org/) for versioning.
For the versions available, see the [tags on this repository](https://github.com/ncotds/nco-lib/tags).

## Developing

Prerequsites:

* [go 1.22+](https://go.dev/doc/install)
* [docker-ce, docker-compose](https://docs.docker.com/engine/install/)
* [pre-commit tool](https://pre-commit.com/#install)

Setup dev environment:

* clone repo and go to the project's root
* setup OMNIbus
  (if you prefer docker,
  see the [repo with Dockerfiles for Netcool](https://github.com/juliusloman/docker-omnibus)
  and example [docker-compose file](docker-compose-omni.yml))
* install tools and enable pre commit hooks:
  ```
  make setup-tools 
  pre-commit install
  ```
* run tests:
  ```
  make lint test
  ```