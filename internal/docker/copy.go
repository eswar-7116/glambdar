package docker

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/moby/moby/client"
)

func copyFileToContainer(
	ctx context.Context,
	cli DockerAPI,
	containerID string,
	srcFile string,
	dstFile string,
) error {
	pr, pw := io.Pipe()

	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		tw := tar.NewWriter(pw)

		info, err := os.Stat(srcFile)
		if err != nil {
			_ = pw.CloseWithError(err)
			errCh <- err
			return
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			_ = pw.CloseWithError(err)
			errCh <- err
			return
		}

		header.Name = filepath.Base(dstFile)

		if err := tw.WriteHeader(header); err != nil {
			_ = pw.CloseWithError(err)
			errCh <- err
			return
		}

		f, err := os.Open(srcFile)
		if err != nil {
			_ = pw.CloseWithError(err)
			errCh <- err
			return
		}

		_, err = io.Copy(tw, f)
		closeErr := f.Close()

		if err == nil {
			err = closeErr
		}

		if err == nil {
			err = tw.Close()
		} else {
			_ = tw.Close()
		}

		if err != nil {
			_ = pw.CloseWithError(err)
			errCh <- err
			return
		}

		errCh <- pw.Close()
	}()

	_, err := cli.CopyToContainer(
		ctx,
		containerID,
		client.CopyToContainerOptions{
			DestinationPath: "/glambdar",
			Content:         pr,
		},
	)

	if err != nil {
		_ = pr.CloseWithError(err)
		return err
	}

	return <-errCh
}

func copyDirToContainer(
	ctx context.Context,
	cli DockerAPI,
	containerID string,
	srcDir string,
	dstDir string,
) error {
	pr, pw := io.Pipe()

	errCh := make(chan error, 1)

	go func() {
		errCh <- func() error {
			tw := tar.NewWriter(pw)

			err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				relPath, err := filepath.Rel(srcDir, path)
				if err != nil {
					return err
				}

				// Don't put the source directory itself into the archive.
				if relPath == "." {
					return nil
				}

				var linkName string

				if info.Mode()&os.ModeSymlink != 0 {
					linkName, err = os.Readlink(path)
					if err != nil {
						return err
					}
				}

				header, err := tar.FileInfoHeader(info, linkName)
				if err != nil {
					return err
				}

				// TAR paths must use '/' regardless of OS.
				header.Name = filepath.ToSlash(relPath)

				if err := tw.WriteHeader(header); err != nil {
					return err
				}

				if info.Mode().IsRegular() {
					f, err := os.Open(path)
					if err != nil {
						return err
					}

					_, copyErr := io.Copy(tw, f)
					closeErr := f.Close()

					if copyErr != nil {
						return copyErr
					}
					if closeErr != nil {
						return closeErr
					}
				}

				return nil
			})

			if err != nil {
				_ = tw.Close()
				_ = pw.CloseWithError(err)
				return err
			}

			if err := tw.Close(); err != nil {
				_ = pw.CloseWithError(err)
				return err
			}

			return pw.Close()
		}()
	}()

	_, copyErr := cli.CopyToContainer(
		ctx,
		containerID,
		client.CopyToContainerOptions{
			DestinationPath: dstDir,
			Content:         pr,
		},
	)

	if copyErr != nil {
		_ = pr.CloseWithError(copyErr)
		return copyErr
	}

	return <-errCh
}
