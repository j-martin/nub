# coding: utf-8
class Bub < Formula
  desc "bub a cli tool for all your bench needs"
  homepage "https://github.com/benchlabs/bub"
  version "0.7.1"
  url "https://s3bucket/contrib/bub-#{version}-darwin-amd64.gz"
  sha256 "9929f95055c00d3146bc4e0ff0beb5429fbf3914786f0f59ac3f6d9924b3bda8"
  def install
    mv "bub-#{version}-darwin-amd64", "bub"
    bin.install "bub"
  end
end
