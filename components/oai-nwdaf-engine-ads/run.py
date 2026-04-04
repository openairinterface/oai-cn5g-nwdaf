# SPDX-License-Identifier: LicenseRef-CSSL-1.0

# This is the main file of oai-nwdaf-engine-ads.

from flask import Flask
from src.config import SERVER_PORT
from src.routes import api

app = Flask(__name__)
app.register_blueprint(api, url_prefix='/')

if __name__ == '__main__':
    app.run(host='0.0.0.0', threaded=True, port=SERVER_PORT,debug=True)
