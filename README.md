# Open Transit: An Implementation of the Mobility Data Specification (MDS)

[![Build Status](https://github.com/technopolitica/open-transit/actions/workflows/ci.yaml/badge.svg?branch=mainline)](https://github.com/technopolitica/open-transit/actions?query=workflow%3ACI+branch%3Amainline++)
[![Go Report Card](https://goreportcard.com/badge/github.com/technopolitica/open-transit)](https://goreportcard.com/report/github.com/technopolitica/open-transit)

Open Transit aims to be a complete implementation of the [Mobility Data Specification (MDS)](https://github.com/openmobilityfoundation/mobility-data-specification/tree/2.0.0) version 2.0, published by the [Open Mobility Foundation](https://www.openmobilityfoundation.org/).

The goals of Open Transit are simple:

- Strict adherence to MDS 2.0 and future supported revisions.
- Backwards compatible with all minor revisions of the current major MDS version, and with the previous major version (except 1.0, which will not be supported).
- Simple, easy deployments with minimal configuration needed for both agencies and providers.

## Status

Open Transit is very much a work in progress. See below for the status of various modules of MDS.

### 🚫 Authentication

Not yet implemented.

### 🚧 [Agency](https://github.com/openmobilityfoundation/mobility-data-specification/blob/2.0.0/agency/README.md)

- **🚧 POST /vehicles:** Basic vehicle registration implemented; many validations and error messages (such as missing params) not yet implemented.
- **🧪 GET /vehicles:** Fully implemented including provider authorization via provider_id claim in JWT bearer token. Some edge cases may not be handled or fully tested.
- **🚧 PUT /vehicles:** Basic vehicle updates implemented including only authorizing providers to update their own vehicles; many validations and error messages (such as missing params) not yet implemented.
- **🚫 GET /vehicles/status:** Not yet implemented.
- **🚫 POST /trips:** Not yet implemented.
- **🚫 POST /telemetry:** Not yet implemented.
- **🚫 POST /events:** Not yet implemented.
- **🚫 POST /stops:** Not yet implemented.
- **🚫 GET /stops:** Not yet implemented.
- **🚫 POST /reports:** Not yet implemented.

### 🚫[Metrics](https://github.com/openmobilityfoundation/mobility-data-specification/blob/2.0.0/agency/README.md)

Not yet implemented.

### 🚫[Provider](https://github.com/openmobilityfoundation/mobility-data-specification/blob/2.0.0/provider/README.md)

Not yet implemented.

### 🚫[Policy](https://github.com/openmobilityfoundation/mobility-data-specification/blob/2.0.0/policy/README.md)

Not yet implemented.

### 🚫[Jurisdiction](https://github.com/openmobilityfoundation/mobility-data-specification/blob/2.0.0/policy/README.md)

Not yet implemented.

### 🚫[Geography](https://github.com/openmobilityfoundation/mobility-data-specification/blob/2.0.0/geography/README.md)

Not yet implemented.
