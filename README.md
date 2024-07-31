# Shared lib for other nco-* tools #

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ncotds_nco-lib&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=ncotds_nco-lib)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=ncotds_nco-lib&metric=coverage)](https://sonarcloud.io/summary/new_code?id=ncotds_nco-lib)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=ncotds_nco-lib&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=ncotds_nco-lib)
[![Build](https://github.com/ncotds/nco-lib/actions/workflows/build-release-assets.yml/badge.svg)](https://github.com/ncotds/nco-lib/actions/workflows/build-release-assets.yml)

> *"Netcool OMNIbus Object Server" - component of IBM Netcool stack, in-memory database to store alerts data*

Repository contains some packages for working with OMNIbus Object Server.

#### Package `dbconnector`

Defines a DB-API interface: 'connector' and 'connection'.
So that your application logic does not depend on a specific implementation of the DB client.
Also contains interface mocks for testing purposes.

```go
package main

import (
  "context"

  "github.com/ncotds/nco-lib/dbconnector"
)

type Pool struct {
	Connector   dbconnector.DBConnector
	Credentials dbconnector.Credentials
	Addr        dbconnector.Addr
	// ...
}

func (p *Pool) Acquire(ctx context.Context) (dbconnector.ExecutorCloser, error) {
	// ... check if there are no idle connections, pool is not full, etc ...
	conn, err := p.Connector.Connect(ctx, p.Addr, p.Credentials)
	if err != nil {
		return nil, err
	}
	// ... store conn in pool
	return conn, nil
}

func (p *Pool) Release(conn dbconnector.ExecutorCloser) error {
	// ... check conn and mark it unused ...
	return nil
}
```

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