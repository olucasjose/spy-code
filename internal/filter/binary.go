// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package filter

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// IsBinaryData avalia o buffer em memória buscando bytes nulos (\x00).
// Esta é a heurística padrão do Git e a mais performática para o Go.
func IsBinaryData(data []byte) bool {
	return bytes.Contains(data, []byte{0})
}

// IsBinaryFile realiza verificação em disco abrindo e lendo estritamente os primeiros 512 bytes.
func IsBinaryFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	return IsBinaryData(buf[:n]), nil
}

// BinaryFilterWriter é um io.Writer customizado projetado para interceptar streams do Git.
// Retém até 512 bytes em buffer; se detectar binário, descarta o restante do fluxo atomicamente,
// evitando poluir o arquivo consolidado (Single-File) com binários não identificados na origem.
type BinaryFilterWriter struct {
	Out          io.Writer
	Header       []byte
	Buffer       bytes.Buffer
	IsBinary     bool
	Determined   bool
	BytesWritten int
	Quiet        bool
	RelPath      string
}

func (w *BinaryFilterWriter) Write(p []byte) (n int, err error) {
	if w.Determined {
		if w.IsBinary {
			return len(p), nil // Drenagem de buffer fantasma (descarte)
		}
		return w.Out.Write(p)
	}

	w.Buffer.Write(p)
	w.BytesWritten += len(p)

	if w.BytesWritten >= 512 {
		w.Determined = true
		w.IsBinary = IsBinaryData(w.Buffer.Bytes())

		if w.IsBinary {
			if !w.Quiet {
				fmt.Printf("  -> Omitido (Binário detectado via Git stream): %s\n", w.RelPath)
			}
			w.Buffer.Reset()
			return len(p), nil
		}

		// Libera a represa (Header + Buffer armazenado)
		if _, err := w.Out.Write(w.Header); err != nil {
			return 0, err
		}
		if _, err := w.Out.Write(w.Buffer.Bytes()); err != nil {
			return 0, err
		}
		w.Buffer.Reset()
	}
	return len(p), nil
}

// Flush consolida o despejo caso o arquivo seja menor que os 512 bytes limitadores do buffer.
func (w *BinaryFilterWriter) Flush() error {
	if !w.Determined {
		w.Determined = true
		w.IsBinary = IsBinaryData(w.Buffer.Bytes())

		if w.IsBinary {
			if !w.Quiet {
				fmt.Printf("  -> Omitido (Binário detectado via Git stream): %s\n", w.RelPath)
			}
			w.Buffer.Reset()
			return nil
		}

		if _, err := w.Out.Write(w.Header); err != nil {
			return err
		}
		if _, err := w.Out.Write(w.Buffer.Bytes()); err != nil {
			return err
		}
		w.Buffer.Reset()
	}
	return nil
}
