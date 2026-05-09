// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package sys

import (
	"os/exec"
	"strings"
	"time"
	_ "time/tzdata" // Embarca o banco de fusos horários da IANA no binário
)

func init() {
	// Se o Go não encontrar o fuso do sistema e cair no fallback (UTC/Local genérico)
	if time.Local.String() == "UTC" || time.Local.String() == "Local" {
		// Tenta ler a propriedade nativa do Android (Termux)
		out, err := exec.Command("getprop", "persist.sys.timezone").Output()
		if err == nil {
			tzName := strings.TrimSpace(string(out))
			if tzName != "" {
				// Usa o tzdata embarcado para mapear o nome da zona para a hora local
				if loc, err := time.LoadLocation(tzName); err == nil {
					time.Local = loc
				}
			}
		}
	}
}
