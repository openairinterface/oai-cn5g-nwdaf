#!/usr/bin/env python3
# SPDX-License-Identifier: MIT

import argparse
import getpass
import json
import logging
import os
import re
import time
import sys
import pexpect
import common.python.cls_cmd as cls_cmd

logging.basicConfig(
    level=logging.DEBUG,
    stream=sys.stdout,
    format="[%(asctime)s] %(levelname)8s: %(message)s"
)

PrivateRegistryURL = 'selfix.sboai.cs.eurecom.fr'
ms_names = ["nbi-analytics", "nbi-events", "nbi-ml", "engine", "engine-ads", "sbi"]
cn_names = ["amf", "ausf", "nrf", "smf", "udm", "udr", "upf"]
cli_image_name = 'nwdaf-cli'

cn5g_deploy_file = 'docker-compose-basic-nrf-ebpf.yaml'
cn5g_conf_file = 'conf/basic_nrf_config_ebpf.yaml'
nwdaf_deploy_file = 'docker-compose-nwdaf-cn-http2.yaml'
original_nwdaf_deploy_contents = None

def _parse_args() -> argparse.Namespace:
    """Parse the command line args

    Returns:
        argparse.Namespace: the created parser
    """
    example_text = '''example:
        ./run-full-test.py --help'''

    parser = argparse.ArgumentParser(description='OAI 5G CORE NETWORK Utility tool',
                                    epilog=example_text,
                                    formatter_class=argparse.RawDescriptionHelpFormatter)

    # NWDAF Micro-Services' Images Tag
    parser.add_argument(
        '--tag', '-t',
        action='store',
        help='NWDAF Micro-Services Images Tag',
    )

    # Pull from local private registry
    parser.add_argument(
        '--pull',
        action='store_true',
        default=False,
        help='Pull from local private registry',
    )

    # Capture PCAP
    parser.add_argument(
        '--capture',
        action='store_true',
        default=False,
        help='Capture',
    )

    args, unknown = parser.parse_known_args()
    return args

def pullFromPrivateRegistry(tag):
    logging.debug(f'\u001B[1mPulling images with tag {tag}\u001B[0m')
    myCmds = cls_cmd.LocalCmd()
    ret = myCmds.run(f'docker login -u oaicicd -p oaicicd {PrivateRegistryURL}')
    if ret.returncode != 0:
        myCmds.close()
        return -1

    for service in ms_names:
        ret = myCmds.run(f'docker pull {PrivateRegistryURL}/oai-nwdaf-{service}:{tag}')
        if ret.returncode != 0:
            myCmds.close()
            return -1

    ret = myCmds.run(f'docker logout {PrivateRegistryURL}')
    if ret.returncode != 0:
        myCmds.close()
        return -1

    myCmds.close()
    return 0

def removePulledImages(tag):
    logging.debug(f'\u001B[1mRemoving images with tag {tag}\u001B[0m')
    myCmds = cls_cmd.LocalCmd()
    for service in ms_names:
        myCmds.run(f'docker rmi {PrivateRegistryURL}/oai-nwdaf-{service}:{tag} || true')

    myCmds.run(f'docker logout {PrivateRegistryURL}')
    myCmds.close()
    return 0

def ensureLocalCliImage(tag):
    myCmds = cls_cmd.LocalCmd()
    existing = myCmds.run(f'docker image inspect {cli_image_name}:{tag}', silent=True)
    if existing.returncode == 0:
        logging.debug(f'\u001B[1mReusing local CLI image with tag {tag}\u001B[0m')
        myCmds.close()
        return 0

    logging.debug(f'\u001B[1mBuilding local CLI image with tag {tag}\u001B[0m')
    ret = myCmds.run(
        f'docker build '
        f'--target {cli_image_name} '
        f'--tag {cli_image_name}:{tag} '
        f'--file cli/Dockerfile '
        f'cli'
    )
    myCmds.close()
    if ret.returncode != 0:
        logging.error('\u001B[1mBuilding the NWDAF CLI image FAILED!\u001B[0m')
    return int(ret.returncode)

def deployOAICN5G():
    logging.debug('\u001B[1mDeploying the OAI CN5G develop\u001B[0m')
    myCmds = cls_cmd.LocalCmd()
    myCmds.run('mkdir -p tests archives')
    if not os.path.isdir('tests/.git'):
        myCmds.run('git clone --branch develop https://gitlab.eurecom.fr/oai/cn5g/oai-cn5g-fed.git tests')
    myCmds.run('cd tests && git submodule update --init ci-scripts')
    myCmds.run(f'sed -i -e "s@enable_usage_reporting: no@enable_usage_reporting: yes@" tests/docker-compose/{cn5g_conf_file}')
    ret = myCmds.run('cd tests/docker-compose && python3 ./core-network.py --type start-basic-ebpf --scenario 1')
    cwd = os.getcwd()
    with open(f'{cwd}/archives/cn5g_deploy.log', 'w') as wfile:
        wfile.write(ret.stdout)
    ret2 = myCmds.run(f'cd tests/docker-compose && docker-compose -f {cn5g_deploy_file} ps --all')
    for line in ret2.stdout.split('\n'):
        logging.debug(line)
    myCmds.close()
    if ret.returncode != 0:
        logging.error('\u001B[1mDeploying the OAI CN5G FAILED!\u001B[0m')
    return int(ret.returncode)

def undeployOAICN5G(do_capture):
    logging.debug('\u001B[1mUn-deploying the OAI CN5G\u001B[0m')
    myCmds = cls_cmd.LocalCmd()
    ret = myCmds.run('docker ps -a | grep oai-amf')
    if ret.returncode != 0:
        logging.debug('No trace of deployed container')
        myCmds.close()
        return 0
    myCmds.run(f'cd tests/docker-compose && docker-compose -f {cn5g_deploy_file} stop')
    for nf in cn_names:
        myCmds.run(f'docker logs oai-{nf} > archives/{nf}.log 2>&1')
    myCmds.run('cd tests/docker-compose && python3 ./core-network.py --type stop-basic-ebpf --scenario 1')
    ret2 = myCmds.run(f'cd tests/docker-compose && docker-compose -f {cn5g_deploy_file} ps --all')
    for line in ret2.stdout.split('\n'):
        logging.debug(line)
    if do_capture:
        for capture_file in ('/tmp/oai-nwdaf.pcap', '/tmp/oai-nwdaf.log'):
            if os.path.exists(capture_file):
                myCmds.run(f'cp {capture_file} archives')
                myCmds.run(f'rm -f {capture_file}')
    myCmds.run('docker volume prune -f')
    myCmds.run('docker network prune -f')
    myCmds.close()
    return 0

def deployNWDAF(tag, do_capture, pulledImages):
    logging.debug('\u001B[1mDeploying the OAI NWDAF micro-services\u001B[0m')
    time.sleep(5)
    global original_nwdaf_deploy_contents
    myCmds = cls_cmd.LocalCmd()
    with open(f'docker-compose/{nwdaf_deploy_file}', 'r') as deploy_file:
        original_nwdaf_deploy_contents = deploy_file.read()
    for service in ms_names:
        if pulledImages:
            myCmds.run(f'sed -i -e "s@oai-nwdaf-{service}:latest@{PrivateRegistryURL}/oai-nwdaf-{service}:{tag}@" docker-compose/{nwdaf_deploy_file}')
        else:
            myCmds.run(f'sed -i -e "s@oai-nwdaf-{service}:latest@oai-nwdaf-{service}:{tag}@" docker-compose/{nwdaf_deploy_file}')
    myCmds.run(f'cp docker-compose/{nwdaf_deploy_file} archives')
    # Deploying in 2 steps, so the nwdaf network is created and we can cpature on it
    myCmds.run(f'docker-compose -f docker-compose/{nwdaf_deploy_file} up -d oai-nwdaf-nbi-gateway')
    captureStatus = 0
    if do_capture:
        time.sleep(5)
        # Using pexpect for background task when passwordless ssh to localhost is configured.
        try:
            myShell = pexpect.spawn(f'ssh {getpass.getuser()}@localhost')
            myShell.timeout = 5
            response = myShell.expect(['Last login', pexpect.EOF, pexpect.TIMEOUT])
            if response == 0:
                time.sleep(1)
                myShell.expect(["\$"])
                capture_cmd = 'nohup sudo tshark -i demo-oai -i cn5g-nwdaf -w /tmp/oai-nwdaf.pcap > /tmp/oai-nwdaf.log 2>&1 &'
                logging.info(capture_cmd)
                myShell.sendline(capture_cmd)
                myShell.expect(["\$", pexpect.EOF, pexpect.TIMEOUT])
                time.sleep(5)
                myShell.sendline('exit')
                for capture_file in ('/tmp/oai-nwdaf.pcap', '/tmp/oai-nwdaf.log'):
                    if os.path.exists(capture_file):
                        myCmds.run(f'chmod 666 {capture_file}')
            else:
                logging.warning(f'Capture requested but ssh startup failed with status {response}; continuing without packet capture')
        except pexpect.exceptions.ExceptionPexpect as err:
            logging.warning(f'Capture requested but could not start packet capture: {err}')
    ret0 = myCmds.run(f'docker-compose -f docker-compose/{nwdaf_deploy_file} up -d')
    # Testing the status of each container
    container_names = ["database", "nbi-gateway", "nbi-analytics", "nbi-events", "nbi-ml", "engine", "engine-ads", "sbi"]
    deployStatus = captureStatus + int(ret0.returncode)
    for container in container_names:
        timeout = 120 if container == "engine-ads" else 30
        ret1 = myCmds.run(f'./tests/ci-scripts/checkContainerStatus.py --container_name oai-nwdaf-{container} --timeout {timeout}')
        deployStatus += int(ret1.returncode)
    ret2 = myCmds.run(f'docker-compose -f docker-compose/{nwdaf_deploy_file} ps --all')
    for line in ret2.stdout.split('\n'):
        logging.debug(line)
    cwd = os.getcwd()
    cli_ret = myCmds.run(
        f'docker run --name test-cli '
        f'--network oai-nwdaf-net '
        f'-e NWDAF_NBI_GATEWAY_URL=http://oai-nwdaf-nbi-gateway:8000 '
        f'-d --volume {cwd}/cli:/nwdaf-cli '
        f'{cli_image_name}:{tag}'
    )
    deployStatus += int(cli_ret.returncode)
    myCmds.close()
    if deployStatus != 0:
        logging.error('\u001B[1mDeploying the OAI NWDAF micro-services FAILED!\u001B[0m')
    return deployStatus

def undeployNWDAF():
    logging.debug('\u001B[1mUn-deploying the OAI NWDAF micro-services\u001B[0m')
    global original_nwdaf_deploy_contents
    myCmds = cls_cmd.LocalCmd()
    myCmds.run('docker rm -f test-cli')
    ret = myCmds.run(f'docker-compose -f docker-compose/{nwdaf_deploy_file} stop')
    for service in ms_names:
        myCmds.run(f'docker logs oai-nwdaf-{service} > archives/nwdaf-{service}.log 2>&1')
    ret = myCmds.run(f'docker-compose -f docker-compose/{nwdaf_deploy_file} down')
    if original_nwdaf_deploy_contents is not None:
        with open(f'docker-compose/{nwdaf_deploy_file}', 'w') as deploy_file:
            deploy_file.write(original_nwdaf_deploy_contents)
        original_nwdaf_deploy_contents = None
    myCmds.close()
    if ret.returncode != 0:
        logging.error('\u001B[1mUn-deploying the OAI NWDAF micro-services FAILED!\u001B[0m')
    return int(ret.returncode)

def testNWDAF():
    testStatus = 0
    logging.debug('\u001B[1mTesting the Analytics Info API\u001B[0m')
    myCmds = cls_cmd.LocalCmd()
    myCmds.run('mkdir -p archives/tests')
    myCmds.run('mkdir -p cli/runtime-requests')

    def prepare_analytics_request(name):
        source_path = f'cli/examples/analytics/{name}.json'
        target_path = f'cli/runtime-requests/{name}.json'
        with open(source_path, 'r') as source_file:
            payload = json.load(source_file)
        payload['ana-req']['startTs'] = '2000-01-01T00:00:00.000Z'
        payload['ana-req']['endTs'] = '2100-01-01T00:00:00.000Z'
        with open(target_path, 'w') as target_file:
            json.dump(payload, target_file, indent=2)
        return f'runtime-requests/{name}.json'

    analyticsTests = []
    analyticsTests.append(('numUe', '.nwPerfs[0].absoluteNum', '0', '1'))
    analyticsTests.append(('numPdu', '.nwPerfs[0].absoluteNum', '0', '1'))
    analyticsTests.append(('ueComm', '.ueComms[0].trafChar.ulVol', '0', '52200'))
    analyticsTests.append(('numPdu_snssais_dnns', '.nwPerfs[0].absoluteNum', '0', '1'))
    # First round of testing (no UEs yet)
    for name, jqPattern, test0Res, test1Res in analyticsTests:
        request_path = prepare_analytics_request(name)
        myCmds.run(f'docker exec test-cli /bin/bash -c "python nwdaf.py analytics {request_path}" > archives/tests/{name}_test0.json 2>&1')
        ret = myCmds.run(f'jq {jqPattern} archives/tests/{name}_test0.json', silent=True)
        if ret.stdout.strip() != test0Res:
            logging.error(f'Test {name} expected was {test0Res}')
            testStatus = -1
    myCmds.run('docker image inspect gnbsim:develop > /dev/null 2>&1 || docker image tag gnbsim:latest gnbsim:develop')
    myCmds.run('cd tests/docker-compose && docker-compose -f docker-compose-gnbsim-ebpf.yaml up -d')
    myCmds.run('./tests/ci-scripts/checkContainerStatus.py --container_name gnbsim-ebpf --timeout 30')
    time.sleep(10)
    # Retrieve the UE allocated IP address
    ret = myCmds.run('docker logs gnbsim-ebpf 2>&1 | grep "UE address:"')
    ipAddr = ''
    if ret.returncode == 0:
        ipRet = re.search('UE address: ([0-9.]+)', ret.stdout)
        if ipRet is not None:
            ipAddr = ipRet.group(1)
            logging.debug(f'UE was allocated this IP address ({ipAddr}) by the CN5G')
    if ipAddr != '':
        myCmds.run(f'docker exec gnbsim-ebpf /bin/bash -c "ping -c50 -i 0.2 -s 1016 -I {ipAddr} 192.168.72.135" > archives/ping_test.log 2>&1')
    time.sleep(20)
    # Second round of testing after connecting one UE and done some traffic
    for name, jqPattern, test0Res, test1Res in analyticsTests:
        request_path = prepare_analytics_request(name)
        myCmds.run(f'docker exec test-cli /bin/bash -c "python nwdaf.py analytics {request_path}" > archives/tests/{name}_test1.json 2>&1')
        ret = myCmds.run(f'jq {jqPattern} archives/tests/{name}_test1.json', silent=True)
        if ret.stdout.strip() != test1Res:
            logging.error(f'Test {name} expected was {test1Res}')
            testStatus = -1
    time.sleep(10)
    myCmds.run('cd tests/docker-compose && docker-compose -f docker-compose-gnbsim-ebpf.yaml stop')
    myCmds.run('docker logs gnbsim-ebpf > archives/gnbsim-ebpf.log 2>&1')
    myCmds.run('cd tests/docker-compose && docker-compose -f docker-compose-gnbsim-ebpf.yaml down')
    if testStatus != 0:
        logging.error('\u001B[1mTesting the Analytics Info API FAILED!\u001B[0m')
    return testStatus

if __name__ == '__main__':
    # Parse the arguments
    args = _parse_args()
    status = 0

    # In CI context, the local register is clean and images shall be pulled
    if args.pull:
        status += pullFromPrivateRegistry(args.tag)

    if status == 0:
        status += ensureLocalCliImage(args.tag)

    # First deploy the core network
    if status == 0:
        status += deployOAICN5G()

    if status == 0:
        status += deployNWDAF(args.tag, args.capture, args.pull)

    if status == 0:
        status += testNWDAF()

    status += undeployNWDAF()
    # Undeploy all the time the core network (even if not deployed)
    status += undeployOAICN5G(args.capture)

    # In CI context, we shall leave test server clean
    if args.pull:
        removePulledImages(args.tag)

    sys.exit(status)
