<!-- SPDX-License-Identifier: CC-BY-4.0 -->

------------------------------------------------------------------------------

                             OPENAIR-CN-5G
 An implementation of the 5G Core network by the OpenAirInterface community.

------------------------------------------------------------------------------

OPENAIR-CN-5G is an implementation of the 3GPP specifications for the 5G Core Network.
At the moment, it contains the following network elements:

* Access and Mobility Management Function (**AMF**)
* Authentication Server Management Function (**AUSF**)
* Location Management Function (**LMF**)
* Network Exposure Function (**NEF**)
* Network Slicing Selection Function (**NSSF**)
* Network Repository Function (**NRF**)
* Network Data Analytics Function (**NWDAF**)
* Policy Control Function (**PCF**)
* Session Management Function (**SMF**)
* Unified Data Management (**UDM**)
* Unified Data Repository (**UDR**)
* Unstructured Data Storage Function (**UDSF**)
* User Plane Function (**UPF**)

Each has its own repository: this repository (`oai-cn5g-nwdaf`) is meant for NWDAF.

## Licence info

The source code is distributed under `Collaborative Standards Software License v1.0 (CSSL v1.0)`.
For more details, visit the [OAI Website](https://openairinterface.org/oai-cssl/).

The full text of `Collaborative Standards Software License v1.0` is also included in the [LICENSE](LICENSE)
file at the root of this repository.

Certain files in the repository are using MIT License and documentation is distributed under
Creative Commons Attribution 4.0 International license.

For third-party softwares, please refer to the [NOTICE](NOTICE) file.

## Where to start

A detailed [tutorial](docs/TUTORIAL.md) is available.

## Collaborative Development

This source code is hosted and maintained on GitHub, enabling collaborative development and contribution:

* Repository: [https://github.com/openairinterface/oai-cn5g-nwdaf](https://github.com/openairinterface/oai-cn5g-nwdaf)

Contribution guidelines and development workflows are described in the [CONTRIBUTING](CONTRIBUTING.md) file.

## Contribution Requests

Anyone is welcome to contribute to any part of the codebase and any network component.

Contributions can include bug fixes, suggestions, design and architecture improvements, as well as feedback on coding and implementation.

## Release Notes

They are available on the [CHANGELOG](CHANGELOG.md) file.

## Repository Structure:

The OpenAirInterface CN NWDAF software is composed of the following parts:

<pre>
openair-cn5g-nwdaf
├── ci-scripts                    : Directory containing the script files for CI framework.
├── cli
│   └── examples
├── components                    : Directory containing the source code divided in micro-services
│   ├── oai-nwdaf-engine
│   ├── oai-nwdaf-engine-ads
│   ├── oai-nwdaf-nbi-analytics
│   ├── oai-nwdaf-nbi-events
│   ├── oai-nwdaf-nbi-ml
│   └── oai-nwdaf-sbi
├── deployment
├── docker-compose
│   └── conf
└── docs
    └── images
</pre>
