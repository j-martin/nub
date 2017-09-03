package main

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	version "github.com/mcuadros/go-version"
)

type S3path struct {
	Region, Bucket, Path string
}

func latestRelease(base S3path) (obj *s3.Object, err error) {
	s3cfg := getAWSConfig(base.Region)
	sess, err := session.NewSession(&s3cfg)
	if err != nil {
		return nil, err
	}
	svc := s3.New(sess)
	results, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &base.Bucket,
		Prefix: &base.Path,
	})
	if err != nil {
		return nil, err
	}

	// regex to find version numbers in pathnames
	versionRegex, err := regexp.Compile("[0-9\\.]+")
	if err != nil {
		return nil, err
	}

	newestVersion := "0.0"
	var newestObj *s3.Object
	for _, obj := range results.Contents {
		if strings.Contains(*obj.Key, runtime.GOOS) {
			currentVersion := string(versionRegex.Find([]byte(*obj.Key)))
			if version.CompareSimple(currentVersion, newestVersion) > 0 {
				newestVersion = currentVersion
				newestObj = obj
			}
		}
	}
	return newestObj, nil
}

func updateBub(path S3path) error {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("Could not get bub's path: %s", err)
	}
	log.Printf("Downloading s3://%s/%s to %s", path.Bucket, path.Path, exe)
	s3cfg := getAWSConfig(path.Region)
	sess, err := session.NewSession(&s3cfg)
	if err != nil {
		return err
	}
	downloader := s3manager.NewDownloader(sess)

	// dl gzipped upstream content to temp file
	fgz, err := ioutil.TempFile("", "bub-update")
	if err != nil {
		return err
	}
	defer fgz.Close()
	defer os.Remove(fgz.Name())
	_, err = downloader.Download(fgz, &s3.GetObjectInput{
		Bucket: &path.Bucket,
		Key:    &path.Path,
	})
	if err != nil {
		return err
	}

	// uncompress to second tempfile
	_, err = fgz.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	f, err := ioutil.TempFile("", "bub-update")
	if err != nil {
		return err
	}
	defer f.Close()
	defer os.Remove(f.Name())

	// transparently gunzip as we download
	gzr, err := gzip.NewReader(fgz)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, gzr)
	if err != nil {
		return err
	}
	if err = os.Chmod(f.Name(), 0755); err != nil {
		return err
	}

	if os.Rename(f.Name(), exe) != nil {
		return err
	}
	log.Printf("Update complete. Run 'bub --version' to be confirm.")
	return nil
}
