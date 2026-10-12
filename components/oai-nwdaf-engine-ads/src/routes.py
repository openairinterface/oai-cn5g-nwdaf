# SPDX-License-Identifier: LicenseRef-CSSL-1.0

# This file contains oai-nwdaf-engine-ads server routes.

from flask import Blueprint, jsonify
from src.config import *
from src.functions import *
import numpy as np
import logging

api = Blueprint('api', __name__)
logging.basicConfig(level=logging.INFO)

@api.route('/abnormal_behaviour/unexpected_large_rate_flow', methods=['GET'])
def handle_unexpected_large_rate_flow_request():
    rows = create_dataframe()
    logging.info([(r['timestamp'], r['value_total']) for r in rows])
    # Get the usefull rows: (hour, uplink volume) of the last seq_dim reports.
    seq = ulrf_scaler.transform([[r['hour'], r['value_ul']] for r in rows[-seq_dim:]])
    # Add model to calculate anomaly probability for sequence.
    seq = np.reshape(seq, (1, seq_dim, num_features))
    predict_seq = ulrf_model.predict(seq)
    mae = np.mean(np.abs(predict_seq[:,:,1:] - seq[:,:,1:]), axis=1)
    distance = np.abs(mae - distance_threshold)
    anomaly_prob = np.minimum(distance / max_distance_threshold, 1)
    logging.info("unexpected_large_rate_flow probability is: %s", anomaly_prob[0][0])
    ratio = int(anomaly_prob[0][0] * 100)
    response_data = {'ratio': ratio}
    # send anomaly probability to client.
    return jsonify(response_data)
