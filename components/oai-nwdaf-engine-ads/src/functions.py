# SPDX-License-Identifier: LicenseRef-CSSL-1.0

# This file contains utils functions.

import pandas as pd
from src.config import *

def add_time_columns(df, timestamp_col):
    df['timestamp'] = pd.to_datetime(df[timestamp_col], unit='s')
    df['year'] = df['timestamp'].dt.year
    df['month'] = df['timestamp'].dt.month
    df['day'] = df['timestamp'].dt.day
    df['hour'] = df['timestamp'].dt.hour
    df['minute'] = df['timestamp'].dt.minute
    return df

def create_dataframe():
    data = []
    for doc in smf_collection.find():
        for qosmon in doc['qosmonlist']:
            data.append({
                "timestamp": qosmon['timestamp'],
                "pduseid": qosmon['pduseid'],
                "value_ul": qosmon['customized_data']['usagereport']['volume']['uplink'] ,
                "value_dl": qosmon['customized_data']['usagereport']['volume']['downlink'],
                "value_total": qosmon['customized_data']['usagereport']['volume']['total']
            })
    # Create a pandas dataframe
    df = pd.DataFrame(data)
    df = add_time_columns(df, 'timestamp')
    return df
