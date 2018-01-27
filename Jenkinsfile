library 'bench-pipeline'

common {
  node('macos') {
    checkoutRepository()
    def env = [
      "GOPATH=${env.WORKSPACE}",
      "PATH=${env.PATH}:/usr/local/go/bin:/${env.HOME}/.go/bin"
    ]
    withEnv(env) {
      commonStage("Build") {
        def workDir = 'src/github.com/benchlabs/bub'
        sh 'git clean -fdx'
        sh "mkdir -p '${workDir}'"
        sh "find . -mindepth 1 -maxdepth 1 -not -name src -not -name pkg -not -name '.git' -exec cp -r '{}' '${workDir}' \\;"
        dir(workDir) {
          sh 'make release'
          sh 'cp -f bin/bub-linux-amd64 /opt/bub/bub'
        }
      }
    }
    tagRepository.pushAll()
  }
}
