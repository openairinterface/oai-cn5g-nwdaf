#!/usr/bin/env python3
# SPDX-License-Identifier: MIT

"""Render test_results_oai_nwdaf.html from the archived docker build logs."""

import os
import re

from common.python.pipeline_args_parse import (
    _parse_args,
)

# Only the header, git info table and footer come from common.
from common.python.html_builder import (
    generate_header,
    generate_footer,
    generate_git_info,
)

REPORT_NAME = 'test_results_oai_nwdaf.html'
SERVICE_NAMES = ['nbi-analytics', 'nbi-events', 'nbi-ml', 'engine', 'engine-ads', 'sbi']

IMAGE_LIST_LOG = 'archives/build/image-list.log'
COLOUR_HEADER = '#33CCFF'
COLOUR_OK = 'lightgreen'
COLOUR_FAILED = 'lightcoral'
COLOUR_SUCCESS = '#008000'
COLOUR_FAIL = '#FF0000'
COLOUR_DETAILS = '#fff5f5'

# Lines worth showing when a build fails.
ERROR_PATTERNS = [
    re.compile(r'^#\d+\s+ERROR:'),
    re.compile(r'^ERROR:'),
    re.compile(r'^#\d+\s+[\d.]+\s+.*\berror\b', re.IGNORECASE),
]
MAX_ERROR_LINES = 6


def escape(text):
    """Escape HTML-significant characters in raw log text."""
    return (text.replace('&', '&amp;')
                .replace('<', '&lt;')
                .replace('>', '&gt;'))


class HtmlReport():
    """Renders test_results_oai_nwdaf.html from the archived build logs."""

    def __init__(self):
        pass

    def generate(self, args):
        """Write the report file."""
        cwd = os.getcwd()
        with open(os.path.join(cwd, REPORT_NAME), 'w') as wfile:
            wfile.write(generate_header(args))
            wfile.write(generate_git_info(args))
            wfile.write(self.nwdafBuildSummary())
            wfile.write(generate_footer())

    def nwdafBuildSummary(self):
        """Build the 'Container Images Build Summary' chapter."""
        services = [self.microserviceDetails(name) for name in SERVICE_NAMES]
        status = all(service['built'] for service in services)
        # Details column only when something failed.
        withDetails = not status

        if status:
            alert = ('<div class="alert alert-success">'
                     '<strong>All Container Target Images were created.</strong></div>')
        else:
            alert = ('<div class="alert alert-danger">'
                     '<strong>Some/All Container Target Images were NOT created.</strong></div>')

        details = '<h2>Container Images Build Summary</h2>' + alert
        details += self.buildTable(services, withDetails)
        return details

    def buildTable(self, services, withDetails):
        """Render the image table and the Final Build Status row."""
        table = '<table class="table-bordered" width="100%" border="1">\n'
        table += f'  <tr bgcolor="{COLOUR_HEADER}">'
        table += '<th style="width:18%">Dockerfile</th>'
        table += '<th>Microservice Image</th>'
        table += '<th style="width:9%">Status</th>'
        table += '<th style="width:10%">Image Size</th>'
        if withDetails:
            table += '<th style="width:32%">Details</th>'
        table += '</tr>\n'

        for service in services:
            if service['built']:
                colour, label = COLOUR_OK, 'OK'
            else:
                colour, label = COLOUR_FAILED, 'FAILED'
            table += '  <tr>'
            table += f'<td><code>{service["dockerfile"]}</code></td>'
            table += f'<td><code>{service["image"]}</code></td>'
            table += f'<td bgcolor="{colour}" align="center"><b>{label}</b></td>'
            table += f'<td align="right">{service["size"]}</td>'
            if withDetails:
                table += f'<td>{self.detailsCell(service)}</td>'
            table += '</tr>\n'

        if withDetails:
            verdictSpan, colour, verdict = 3, COLOUR_FAIL, 'FAIL'
        else:
            verdictSpan, colour, verdict = 2, COLOUR_SUCCESS, 'SUCCESS'
        table += f'  <tr><th bgcolor="{COLOUR_HEADER}" colspan="2">Final Build Status</th>'
        table += (f'<th bgcolor="{colour}" colspan="{verdictSpan}" align="left">'
                  f'<font color="white">{verdict}</font></th></tr>\n')
        table += '</table>\n'
        return table

    def detailsCell(self, service):
        """Render a failed image's error lines."""
        if service['built'] or not service['details']:
            return ''
        body = escape('\n'.join(service['details']))
        return (f'<pre style="border:none;background-color:{COLOUR_DETAILS};margin:0;'
                'white-space:pre-wrap;word-break:break-word">'
                f'{body}</pre>')

    def microserviceDetails(self, serviceName):
        """Read one microservice's build log."""
        cwd = os.getcwd()
        image = f'oai-nwdaf-{serviceName}'
        info = {
            'dockerfile': f'Dockerfile.{serviceName}',
            'image': image,
            'built': False,
            'size': '-',
            'details': [],
        }

        logFileName = f'{cwd}/archives/build/{serviceName}.log'
        if not os.path.isfile(logFileName):
            info['details'] = ['no build log produced']
            return info

        imageTag = ''
        errors = []
        with open(logFileName, 'r') as logfile:
            for line in logfile:
                line = line.rstrip('\n')
                result = re.search(
                    f'naming to docker.io/library/{image}:([0-9a-zA-Z\\-\\_\\.]+)', line)
                if result is not None:
                    info['built'] = True
                    imageTag = result.group(1)
                if any(pattern.search(line) for pattern in ERROR_PATTERNS):
                    errors.append(line.strip())

        if imageTag:
            info['image'] = f'{image}:{imageTag}'
            info['size'] = self.imageSize(image, imageTag)
        if not info['built']:
            # The last errors are the ones that stopped the build.
            info['details'] = errors[-MAX_ERROR_LINES:] or ['build did not produce an image']

        return info

    def imageSize(self, image, imageTag):
        """Read the image size from `docker image ls`."""
        cwd = os.getcwd()
        listFile = os.path.join(cwd, IMAGE_LIST_LOG)
        if not os.path.isfile(listFile):
            return '-'
        with open(listFile, 'r') as logfile:
            for line in logfile:
                if (re.search(f'{image}:{imageTag}(\\s|$)', line) is None and
                        re.search(f'{image}\\s+{imageTag}(\\s|$)', line) is None):
                    continue
                for token in line.split():
                    result = re.fullmatch(r'([0-9]+(?:\.[0-9]+)?)([KMGT]?B)', token)
                    if result is not None and result.group(1) != '0':
                        return f'{result.group(1)} {result.group(2)}'
        return '-'

    def appendToTestReports(self, args):
        """Add the git info table to any core-network test report."""
        gitInfo = generate_git_info(args)
        cwd = os.getcwd()
        for reportFile in os.listdir(cwd):
            if reportFile.endswith('.html') and re.search('results_oai_cn5g_', reportFile) is not None:
                newFile = ''
                gitInfoAppended = False
                with open(os.path.join(cwd, reportFile), 'r') as rfile:
                    for line in rfile:
                        if re.search('<h2>', line) is not None and not gitInfoAppended:
                            gitInfoAppended = True
                            newFile += gitInfo
                        newFile += line
                with open(os.path.join(cwd, reportFile), 'w') as wfile:
                    wfile.write(newFile)


if __name__ == '__main__':
    # Parse the arguments
    args = _parse_args()

    # Generate report
    HTML = HtmlReport()
    HTML.generate(args)
    HTML.appendToTestReports(args)
