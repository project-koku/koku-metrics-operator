package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/crhchttp"
	"github.com/project-koku/koku-metrics-operator/dirconfig"
	"github.com/project-koku/koku-metrics-operator/packaging"
)

func packageFiles(p *packaging.FilePackager) {
	log := log.WithName("packageAndUpload")

	// if its time to package
	if !checkCycle(log, *p.CR.Status.Upload.UploadCycle, p.CR.Status.Packaging.LastSuccessfulPackagingTime, "file packaging") {
		return
	}

	// Package and split the payload if necessary
	p.CR.Status.Packaging.PackagingError = ""
	if err := p.PackageReports(); err != nil {
		log.Error(err, "PackageReports failed")
		// update the CR packaging error status
		p.CR.Status.Packaging.PackagingError = err.Error()
	}
}

func uploadFiles(r *MetricsConfigReconciler, authConfig *crhchttp.AuthConfig, cr *metricscfgv1beta1.MetricsConfig, dirCfg *dirconfig.DirectoryConfig, packager *packaging.FilePackager) error {
	log := log.WithName("uploadFiles")

	// if its time to upload/package
	if !*cr.Spec.Upload.UploadToggle {
		log.Info("operator is configured to not upload reports")
		return nil
	}
	if !checkCycle(log, *cr.Status.Upload.UploadCycle, cr.Status.Upload.LastSuccessfulUploadTime, "upload") {
		return nil
	}

	uploadFiles, err := dirCfg.Upload.GetFiles()
	if err != nil {
		log.Error(err, "failed to read upload directory")
		return err
	}

	if len(uploadFiles) <= 0 {
		log.Info("no files to upload")
		return nil
	}

	log.Info("files ready for upload: " + strings.Join(uploadFiles, ", "))
	log.Info("pausing for " + fmt.Sprintf("%d", *cr.Status.Upload.UploadWait) + " seconds before uploading")
	time.Sleep(time.Duration(*cr.Status.Upload.UploadWait) * time.Second)
	for _, file := range uploadFiles {
		if !strings.Contains(file, "tar.gz") {
			continue
		}

		manifestInfo, err := packager.GetFileInfo(filepath.Join(dirCfg.Upload.Path, file))
		if err != nil {
			log.Error(err, "Could not read file information from tar.gz")
			continue
		}

		log.Info(fmt.Sprintf("uploading file: %s", file))
		// grab the body and the multipart file header
		body, contentType, err := crhchttp.GetMultiPartBodyAndHeaders(filepath.Join(dirCfg.Upload.Path, file))
		if err != nil {
			log.Error(err, "failed to set multipart body and headers")
			return err
		}
		ingressURL := cr.Status.APIURL + cr.Status.Upload.IngressAPIPath
		uploadStatus, uploadTime, requestID, err := crhchttp.Upload(authConfig, contentType, "POST", ingressURL, body, manifestInfo, file)
		cr.Status.Upload.LastUploadStatus = uploadStatus
		cr.Status.Upload.LastPayloadName = file
		cr.Status.Upload.LastPayloadFiles = manifestInfo.Files
		cr.Status.Upload.LastPayloadManifestID = manifestInfo.UUID
		cr.Status.Upload.LastPayloadRequestID = requestID
		cr.Status.Upload.UploadError = ""
		if err != nil {
			log.Error(err, "upload failed")
			cr.Status.Upload.UploadError = err.Error()
			return nil
		}
		if strings.Contains(uploadStatus, "202") {
			cr.Status.Upload.LastSuccessfulUploadTime = uploadTime
			// remove the tar.gz after a successful upload
			log.Info("removing tar file since upload was successful")
			if err := os.Remove(filepath.Join(dirCfg.Upload.Path, file)); err != nil {
				log.Error(err, "error removing tar file")
			}
		}
	}
	return nil
}
