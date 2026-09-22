#!/usr/bin/env python3
"""Synthetic HTTP smoke checks; optional response-schema validation in CI/dev venv."""
import argparse
import json
import os
import urllib.request
import urllib.error

parser = argparse.ArgumentParser()
parser.add_argument('--url', default='http://127.0.0.1:8080')
parser.add_argument('--schema', action='store_true')
args = parser.parse_args()
token = os.environ.get('KRIPA_API_TOKEN', '')
if os.environ.get('KRIPA_API_TOKEN_FILE'):
    with open(os.environ['KRIPA_API_TOKEN_FILE']) as f:
        token = f.read().strip()
spec = None
if args.schema:
    import yaml
    import jsonschema
    with open('api/openapi.yaml') as f:
        spec = yaml.safe_load(f)
    # Convert OpenAPI 3.0 nullable to JSON Schema's union type for instance checks.
    def convert(value):
        if isinstance(value, dict):
            if value.pop('nullable', False):
                value['type'] = [value['type'], 'null']
                if 'enum' in value:
                    value['enum'].append(None)
            for child in value.values():
                convert(child)
        elif isinstance(value, list):
            for child in value:
                convert(child)
    convert(spec)

def call(path, method='GET', payload=None, expected=200):
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = 'Bearer ' + token
    req = urllib.request.Request(args.url + path, data=None if payload is None else json.dumps(payload).encode(), headers=headers, method=method)
    try:
        response = urllib.request.urlopen(req, timeout=15)
    except urllib.error.HTTPError as exc:
        response = exc
    with response:
        body = response.read()
        assert response.status == expected, (path, response.status, body)
        assert response.headers['X-Request-ID']
        if path == '/metrics':
            assert b'kripa_native_queue_wait_seconds_total' in body
            return
        value = json.loads(body)
        if spec:
            schema = spec['paths'][path][method.lower()]['responses'][str(expected)]['content']['application/json']['schema']
            jsonschema.Draft4Validator(dict(schema, components=spec['components']), format_checker=jsonschema.FormatChecker()).validate(value)
        return value

call('/health/live')
call('/health/ready')
call('/v1/meta')
chart = dict(date='2000-01-01', time='12:00', timezone='UTC', time_status='exact', latitude=51.5, longitude=0, profile='western_tropical_v1')
assert len(call('/v1/charts', 'POST', chart)['planets']) == 13
chart.pop('time'); chart['time_status'] = 'unknown'
assert call('/v1/charts', 'POST', chart)['ascendant'] is None
panchang = dict(date='2026-09-21', timezone='Asia/Kolkata', latitude=12.9716, longitude=77.5946, profile='lahiri_upper_limb_v1')
assert call('/v1/panchang', 'POST', panchang)['review_status'] == 'astronomical_preview'
call('/v1/panchang', 'POST', panchang)
call('/v1/charts', 'POST', dict(chart, unexpected=True), 400)
call('/v1/charts', 'POST', dict(chart, date='2024-02-30'), 422)
call('/metrics')
print('HTTP smoke passed' + ('; response schemas validated' if spec else ''))
