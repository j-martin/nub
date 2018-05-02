library 'bench-pipeline'

common {
  node('macos') {
    checkoutRepository()
    def env = [
      "PATH=/usr/local/bin/:${env.PATH}",
      "GOPATH=/private${pwd()}"
    ]
    withCredentials([string(credentialsId: 'bub-bucket', variable: 'S3_BUCKET')]) {
      withEnv(env) {
        commonStage("Build") {
          def workDir = 'src/github.com/j-martin/bub'
          sh 'git clean -fdx'
          sh "mkdir -p '${workDir}'"
          sh "find . -mindepth 1 -maxdepth 1 -not -name src -not -name pkg -not -name '.git' -exec cp -r '{}' '${workDir}' \\;"
          dir(workDir) {
            sh 'make release'
            stash includes: 'bin/*', name: 'binaries'
          }
        }
      }
      tagRepository.pushAll()
    }
  }
  node('master') {
    unstash 'binaries'
    sh 'cp -f bin/bub-linux-amd64 /opt/bub/bub'
  }
}
