pipeline {
    agent any
    environment {
        SLACK_CHANNEL = 'jenkins'
        SLACK_TEAM_DOMAIN = 'phragmites'
        TEST_NAME = sh(script: 'echo ${BRANCH_NAME} | sed "s/[^A-Za-z0-9\\-]*//g" | tr "[:upper:]" "[:lower:]"', returnStdout: true).trim()
        ETH_TEST_DATA_DIR='/etherium-test-data/data'
        ETH_TEST_RESULTS_BASE_DIR='/etherium-test-data/tests-results'
        DOCKER_GO_IMAGE='golang:1.12'
    }
    options {
      ansiColor('xterm')
      disableConcurrentBuilds()
    }
    stages {
        stage('Go Tests') {
            steps {
                sh '''
                    # Hack, get right mount point
                    WORKSPACE_FIXED=$(echo $WORKSPACE | sed 's,/var/jenkins_home/,/var/lib/jenkins/,g')
                    docker run --rm --name go-evm-${TEST_NAME} \
                        -v ${WORKSPACE_FIXED}:/app \
                        -v ${ETH_TEST_DATA_DIR}:${ETH_TEST_DATA_DIR}:ro \
                        -v ${ETH_TEST_RESULTS_BASE_DIR}/${TEST_NAME}:${ETH_TEST_RESULTS_BASE_DIR}/${TEST_NAME} \
                        -e ETH_TEST_DATA_DIR=${ETH_TEST_DATA_DIR} \
                        -e ETH_TEST_RESULTS_DIR=${ETH_TEST_RESULTS_BASE_DIR}/${TEST_NAME} \
                        -w /app \
                        ${DOCKER_GO_IMAGE} \
                        go test
                '''
            }
            post {
                always {
                    sh 'docker kill go-evm-${TEST_NAME} || true'
                }
            }
        }
    }
post {
    success {
      slackSend (channel: "${SLACK_CHANNEL}", teamDomain: "${SLACK_TEAM_DOMAIN}", tokenCredentialId: 'SLACK_TOKEN_ID',
                color: '#00FF00', message: "SUCCESSFUL: Job '${JOB_NAME} [${BUILD_NUMBER}]' (${BUILD_URL})")
    }
    failure {
      slackSend (channel: "${SLACK_CHANNEL}", teamDomain: "${SLACK_TEAM_DOMAIN}", tokenCredentialId: 'SLACK_TOKEN_ID',
                color: '#FF0000', message: "FAILED: Job '${JOB_NAME} [${BUILD_NUMBER}]' (${BUILD_URL})")
    }
  }
}
