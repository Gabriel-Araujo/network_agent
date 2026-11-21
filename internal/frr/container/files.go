package container

import (
	"fmt"
	"os"
	"strings"
)

// DefaultVtyshConf enables integrated FRR configuration so vtysh operates over
// /etc/frr/frr.conf consistently across daemons.
const DefaultVtyshConf = "service integrated-vtysh-config\n"

// AllDaemons lists the 17 optional FRR 10.5 daemons enabled in /etc/frr/daemons
// (excluding the test daemon sharpd). Together with the 4 always-on daemons
// (watchfrr, mgmtd, zebra, staticd), enabling this list starts all 21 daemons.
var AllDaemons = []string{
	"bgpd",
	"ospfd",
	"ospf6d",
	"ripd",
	"ripngd",
	"isisd",
	"pimd",
	"pim6d",
	"ldpd",
	"nhrpd",
	"eigrpd",
	"babeld",
	"pbrd",
	"bfdd",
	"fabricd",
	"vrrpd",
	"pathd",
}

// optionalDaemonOrder mirrors /etc/frr/daemons in quay.io/frrouting/frr:10.5.0.
var optionalDaemonOrder = []string{
	"bgpd",
	"ospfd",
	"ospf6d",
	"ripd",
	"ripngd",
	"isisd",
	"pimd",
	"pim6d",
	"ldpd",
	"nhrpd",
	"eigrpd",
	"babeld",
	"sharpd",
	"pbrd",
	"bfdd",
	"fabricd",
	"vrrpd",
	"pathd",
}

const daemonOptionsBlock = `vtysh_enable=yes
zebra_options="  -A 127.0.0.1 -s 90000000"
mgmtd_options="  -A 127.0.0.1"
bgpd_options="   -A 127.0.0.1 -M rpki -M bmp"
ospfd_options="  -A 127.0.0.1"
ospf6d_options=" -A ::1"
ripd_options="   -A 127.0.0.1"
ripngd_options=" -A ::1"
isisd_options="  -A 127.0.0.1"
pimd_options="   -A 127.0.0.1"
pim6d_options="  -A ::1"
ldpd_options="   -A 127.0.0.1"
nhrpd_options="  -A 127.0.0.1"
eigrpd_options=" -A 127.0.0.1"
babeld_options=" -A 127.0.0.1"
sharpd_options=" -A 127.0.0.1"
pbrd_options="   -A 127.0.0.1"
staticd_options="-A 127.0.0.1"
bfdd_options="   -A 127.0.0.1"
fabricd_options="-A 127.0.0.1"
vrrpd_options="  -A 127.0.0.1"
pathd_options="  -A 127.0.0.1"
MAX_FDS=4096
`

// RenderDaemons generates an /etc/frr/daemons file with LF line endings,
// enabling only the listed optional daemons (watchfrr, mgmtd, zebra, and
// staticd are always started by FRR). bgpd is always configured with -M rpki
// -M bmp and MAX_FDS=4096.
func RenderDaemons(enabled []string) string {
	want := make(map[string]bool, len(enabled))
	for _, d := range enabled {
		name := strings.TrimSpace(d)
		if name != "" {
			want[name] = true
		}
	}

	var b strings.Builder
	for _, d := range optionalDaemonOrder {
		state := "no"
		if want[d] {
			state = "yes"
		}
		b.WriteString(d)
		b.WriteByte('=')
		b.WriteString(state)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(daemonOptionsBlock)
	return b.String()
}

// NormalizeLF converts CRLF and bare CR line endings to LF so files mounted
// from Windows hosts are parsed cleanly by FRR and POSIX sh inside Linux
// containers.
func NormalizeLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

// WriteLF writes content to path with LF line endings.
func WriteLF(path, content string, perm os.FileMode) error {
	if err := os.WriteFile(path, []byte(NormalizeLF(content)), perm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
