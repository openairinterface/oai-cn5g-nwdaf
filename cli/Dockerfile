# SPDX-License-Identifier: LicenseRef-CSSL-1.0

FROM python:3.6.9-slim AS nwdaf-cli

COPY ./requirements.txt /tmp
RUN pip install --upgrade "pip<22" "setuptools<60" "wheel<0.38" && \
    pip install -r /tmp/requirements.txt && \
    rm -f /tmp/requirements.txt

WORKDIR /nwdaf-cli
CMD ["sleep", "infinity"]
