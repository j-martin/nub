commonNode {
  checkout scm
  for (cause in currentBuild.rawBuild.getCauses()) {
    if (cause instanceof Cause.UserIdCause) {
      println(cause.getUserName())
    } else if (cause instanceof Cause.UserCause) {
      println('userCause')
    } else if (cause instanceof Cause.UpstreamCause) {
      println('upstrema')
    } else {
      println(cause)
    }
  }
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
