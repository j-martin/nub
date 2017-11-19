library 'bench-pipeline'

commonNode {
  checkout scm
  def env = [
    "GOPATH=${env.WORKSPACE}",
    "PATH=${env.PATH}:/usr/local/go/bin:/${env.HOME}/.go/bin"
  ]
  withEnv(env) {
    commonStage("Build") {
      sh 'git clean -fdx'
      sh 'ln -f -s "$PWD" "$PWD/src/bub"'
      dir('src/bub') {
        sh 'make release'
        sh 'cp -f bin/bub-linux-amd64 /opt/bub/bub'
      }
    }
  }
  tagRepository.pushAll()
}
