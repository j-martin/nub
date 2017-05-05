commonNode {
  checkout scm

  tagRepository('test', '', true)

  def env = [
    "GOPATH=${env.HOME}/.go",
    "PATH=${env.PATH}:/usr/local/go/bin:/${env.HOME}/.go/bin"
  ]
  withEnv(env) {
    stage("Build") {
    sh "make deps release"
    sh "cp -f bin/bub-linux-amd64 /opt/bub/bub"
    }
  }
  tagRepository()
}
