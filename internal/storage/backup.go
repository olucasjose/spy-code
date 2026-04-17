// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"go.etcd.io/bbolt"
)

// BackupSchema define a estrutura do JSON exportado. Migrado para a camada base.
type BackupSchema struct {
	RepoID       string               `json:"repo_id"`
	RepoName     string               `json:"repo_name,omitempty"`
	RepoDenylist []string             `json:"repo_denylist,omitempty"`
	Tags         map[string]TagBackup `json:"tags,omitempty"`
}

type TagBackup struct {
	Meta    TagMeta  `json:"meta"`
	Files   []string `json:"files,omitempty"`
	Ignored []string `json:"ignored,omitempty"`
}

// DumpGitRepositoryData extrai do banco todos os dados atrelados a um repositório Git específico.
func DumpGitRepositoryData(repoID string) (BackupSchema, error) {
	db, err := Open()
	if err != nil {
		return BackupSchema{}, err
	}
	defer db.Close()

	backup := BackupSchema{
		RepoID: repoID,
		Tags:   make(map[string]TagBackup),
	}

	err = db.View(func(tx *bbolt.Tx) error {
		// Puxa Denylist do Repo
		if gitIgnoredBucket := tx.Bucket([]byte(BucketGitIgnored)); gitIgnoredBucket != nil {
			if repoBucket := gitIgnoredBucket.Bucket([]byte(repoID)); repoBucket != nil {
				_ = repoBucket.ForEach(func(k, v []byte) error {
					backup.RepoDenylist = append(backup.RepoDenylist, string(k))
					return nil
				})
			}
		}

		// Puxa Tags do Repo
		tagsBucket := tx.Bucket([]byte(BucketTags))
		filesBucket := tx.Bucket([]byte(BucketFiles))
		ignoredBucket := tx.Bucket([]byte(BucketIgnored))

		if tagsBucket != nil {
			_ = tagsBucket.ForEach(func(k, v []byte) error {
				meta := ParseTagMeta(v)
				if meta.Type != TagTypeGit || meta.RepoID != repoID {
					return nil // Ignora tags locais ou de outros repos
				}

				if backup.RepoName == "" && meta.RepoName != "" {
					backup.RepoName = meta.RepoName
				}

				tagName := string(k)
				tb := TagBackup{Meta: meta}

				if filesBucket != nil {
					if pb := filesBucket.Bucket(k); pb != nil {
						_ = pb.ForEach(func(fk, fv []byte) error {
							tb.Files = append(tb.Files, string(fk))
							return nil
						})
					}
				}
				if ignoredBucket != nil {
					if pb := ignoredBucket.Bucket(k); pb != nil {
						_ = pb.ForEach(func(ik, iv []byte) error {
							tb.Ignored = append(tb.Ignored, string(ik))
							return nil
						})
					}
				}

				backup.Tags[tagName] = tb
				return nil
			})
		}
		return nil
	})

	return backup, err
}

// RestoreGitRepositoryData limpa e injeta atomicamente os dados do backup de volta no banco.
func RestoreGitRepositoryData(currentGitRoot string, backup BackupSchema) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		// Restaura Denylist
		if len(backup.RepoDenylist) > 0 {
			gitIgnoredBucket := tx.Bucket([]byte(BucketGitIgnored))
			repoBucket, err := gitIgnoredBucket.CreateBucketIfNotExists([]byte(backup.RepoID))
			if err != nil {
				return err
			}
			for _, p := range backup.RepoDenylist {
				if err := repoBucket.Put([]byte(p), []byte("1")); err != nil {
					return err
				}
			}
		}

		// Restaura Tags
		if len(backup.Tags) > 0 {
			tagsBucket := tx.Bucket([]byte(BucketTags))
			filesBucket := tx.Bucket([]byte(BucketFiles))
			ignoredBucket := tx.Bucket([]byte(BucketIgnored))

			for tagName, tagData := range backup.Tags {
				meta := tagData.Meta
				meta.GitRoot = currentGitRoot

				if err := tagsBucket.Put([]byte(tagName), EncodeTagMeta(meta)); err != nil {
					return err
				}

				projFiles, err := filesBucket.CreateBucketIfNotExists([]byte(tagName))
				if err != nil {
					return err
				}
				for _, p := range tagData.Files {
					if err := projFiles.Put([]byte(p), []byte("1")); err != nil {
						return err
					}
				}

				projIgnored, err := ignoredBucket.CreateBucketIfNotExists([]byte(tagName))
				if err != nil {
					return err
				}
				for _, p := range tagData.Ignored {
					if err := projIgnored.Put([]byte(p), []byte("1")); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}
