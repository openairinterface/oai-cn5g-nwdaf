# SPDX-License-Identifier: LicenseRef-CSSL-1.0

# This file contains utils functions.

from datetime import datetime, timezone
from src.config import *

def create_dataframe():
    """Usage reports from the SMF collection, oldest first, one dict per report."""
    data = []
    for doc in smf_collection.find():
        for qosmon in doc['qosmonlist']:
            ts = float(qosmon['timestamp'])
            data.append({
                "timestamp": ts,
                "hour": datetime.fromtimestamp(ts, timezone.utc).hour,
                "pduseid": qosmon['pduseid'],
                "value_ul": qosmon['customized_data']['usagereport']['volume']['uplink'] ,
                "value_dl": qosmon['customized_data']['usagereport']['volume']['downlink'],
                "value_total": qosmon['customized_data']['usagereport']['volume']['total']
            })
    return data
