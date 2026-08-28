# SPDX-License-Identifier: LicenseRef-CSSL-1.0

FROM python:3.14.7-slim AS nwdaf-cli

COPY ./requirements.txt /tmp
RUN pip install --upgrade pip setuptools wheel && \
    pip install -r /tmp/requirements.txt && \
    rm -f /tmp/requirements.txt

WORKDIR /nwdaf-cli
CMD ["sleep", "infinity"]
