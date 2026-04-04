# SPDX-License-Identifier: LicenseRef-CSSL-1.0

import json
import os
import requests
import typer
from flask import Flask, request

nbi_gateway_url = os.environ.get('NWDAF_NBI_GATEWAY_URL', 'http://oai-nwdaf-nbi-gateway')
analytics_url = f'{nbi_gateway_url}/nnwdaf-analyticsinfo/v1/analytics'
subscription_url = f'{nbi_gateway_url}/nnwdaf-eventssubscription/v1/subscriptions'
app = typer.Typer()
flask_app = Flask(__name__)

# Command for analytics
@app.command()
def analytics(
    json_file: str = typer.Argument(..., help="JSON analytics request file name"),
):
    """
    Perform analytics based on the provided JSON request file.
    """
    with open(json_file) as file:
        data = json.load(file)
    # Extract parameters from JSON
    event_id = data['event-id']
    ana_req = json.dumps(data['ana-req'])
    event_filter = json.dumps(data['event-filter'])
    supported_features = json.dumps(data['supported-features'])
    tgt_ue = json.dumps(data['tgt-ue'])
    params = {
        'event-id': event_id,
        'ana-req': ana_req,
        'event-filter': event_filter,
        'supported-features': supported_features,
        'tgt-ue': tgt_ue
    }
    response = requests.get(analytics_url, params=params)
    parsed_response = json.loads(response.text)
    typer.echo(json.dumps(parsed_response, indent=4))

# Command for subscribing to events
@app.command()
def subscribe(
    subscription_file: str = typer.Argument(..., help="JSON subscription file name"),
):
    """
    Subscribe to events based on the provided JSON subscription file.
    """
    with open(subscription_file) as f:
        subscription_data = json.load(f)
    response = requests.post(subscription_url, json=subscription_data)
    # Handle response if necessary

    @flask_app.route('/notification', methods=['POST'])
    def handle_notification():
        """
        Endpoint to handle incoming notifications.
        """
        data = request.json
        notif_data = data
        typer.echo(f'\nReceived notification for event: {notif_data["event"]}')
        typer.echo(json.dumps(notif_data, indent=4))
        return 'OK'

    flask_app.run(host='0.0.0.0', port=3000, debug=True)

if __name__ == '__main__':
    app()
