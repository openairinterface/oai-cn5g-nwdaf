# SPDX-License-Identifier: LicenseRef-CSSL-1.0

# oai-nwdaf-engine-ads configuration parameters.

import os
from pymongo import MongoClient
from src.autoencoder import KerasAutoencoder, load_scaler

# Env Variables
SERVER_PORT = os.environ.get('SERVER_PORT','8989')
MONGODB_URI = os.environ.get('MONGODB_URI','mongodb://localhost:27017')
NWDAF_DATABASE_NAME = os.environ.get('MONGODB_DATABASE_NAME', 'testing')
MONGODB_COLLECTION_NAME_AMF = os.environ.get('MONGODB_COLLECTION_NAME_AMF', 'amf')
MONGODB_COLLECTION_NAME_SMF = os.environ.get('MONGODB_COLLECTION_NAME_SMF', 'smf')

# Global variables
client = MongoClient(MONGODB_URI)
nwdaf_db = client[NWDAF_DATABASE_NAME]
amf_collection = nwdaf_db[MONGODB_COLLECTION_NAME_AMF]
smf_collection = nwdaf_db[MONGODB_COLLECTION_NAME_SMF]

# Autoencoder parameters
ulrf_model = KerasAutoencoder('models/unexpected_large_rate_flow/model.h5')
ulrf_scaler = load_scaler('models/unexpected_large_rate_flow/scaler.pkl')
seq_dim = 12
num_features = 2
distance_threshold = 0.26
max_distance_threshold = 2
