package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var diffZipCmd = &cobra.Command{
	Use:   "diff-zip <commit1> <commit2>",
	Short: "Compacta arquivos alterados entre dois commits do Git no disco",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		commit1 := args[0]
		commit2 := args[1]

		fmt.Printf("Comparando %s -> %s\n\n", commit1, commit2)

		files := getChangedFiles(commit1, commit2)
		createZip(files)
	},
}

func init() {
	rootCmd.AddCommand(diffZipCmd)
}

func getChangedFiles(c1, c2 string) []string {
	cmd := exec.Command("git", "diff", "--name-status", c1, c2)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao executar git diff. Certifique-se de que está em um repositório Git válido.\n%s\n", stderr.String())
		os.Exit(1)
	}

	var filesToZip []string
	lines := strings.Split(out.String(), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		statusCode := strings.ToUpper(parts[0])
		statusChar := statusCode[0]

		var filePath string
		isRename := false

		if (statusChar == 'A' || statusChar == 'M') && len(parts) >= 2 {
			filePath = parts[1]
		} else if statusChar == 'R' && len(parts) >= 3 {
			filePath = parts[2]
			isRename = true
		} else {
			continue
		}

		info, err := os.Stat(filePath)
		if err == nil && !info.IsDir() {
			filesToZip = append(filesToZip, filePath)
			if isRename {
				fmt.Printf("  R: %s (renomeado)\n", filePath)
			} else {
				fmt.Printf("  %c: %s\n", statusChar, filePath)
			}
		}
	}

	return filesToZip
}

func createZip(files []string) {
	if len(files) == 0 {
		fmt.Println("\nNenhum arquivo encontrado para compactar no disco.")
		return
	}

	fmt.Printf("\n%d arquivo(s) para compactar.\n", len(files))

	timestamp := time.Now().Format("20060102_150405")
	zipName := fmt.Sprintf("git_changes_%s.zip", timestamp)

	fmt.Printf("Criando %s...\n", zipName)

	zipFile, err := os.Create(zipName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nErro de I/O ao criar o arquivo zip: %v\n", err)
		os.Exit(1)
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for _, file := range files {
		if err := addFileToZip(archive, file); err != nil {
			fmt.Fprintf(os.Stderr, "Aviso: falha ao adicionar %s ao zip: %v\n", file, err)
		}
	}

	fmt.Printf("\nSucesso! %s criado com %d arquivo(s).\n", zipName, len(files))
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = filename
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, fileToZip)
	return err
}
