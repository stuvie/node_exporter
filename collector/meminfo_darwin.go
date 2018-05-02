// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !nomeminfo

package collector

// #include <mach/mach_host.h>
import "C"

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/prometheus/common/log"
	"golang.org/x/sys/unix"
)

func (c *meminfoCollector) getMemInfo() (map[string]float64, error) {
	infoCount := C.mach_msg_type_number_t(C.HOST_VM_INFO_COUNT)
	vmstat := C.vm_statistics_data_t{}
	ret := C.host_statistics(
		C.host_t(C.mach_host_self()),
		C.HOST_VM_INFO,
		C.host_info_t(unsafe.Pointer(&vmstat)),
		&infoCount,
	)
	if ret != C.KERN_SUCCESS {
		return nil, fmt.Errorf("Couldn't get memory statistics, host_statistics returned %d", ret)
	}
	totalb, err := unix.Sysctl("hw.memsize")
	if err != nil {
		return nil, err
	}
	// Syscall removes terminating NUL which we need to cast to uint64
	total := binary.LittleEndian.Uint64([]byte(totalb + "\x00"))

	// aggregate some memory stats
	available := vmstat.free_count + vmstat.inactive_count + vmstat.purgeable_count
	gig := 1073741824.0

	ps := C.natural_t(syscall.Getpagesize())
	/*
		log.Infof("****")
		log.Debugf("darwin mem ps: %d", ps)
		log.Infof("darwin mem total: %.3f Gig", float64(total)/gig)
		log.Infof("darwin mem inactive: %.3f Gig", float64(ps*vmstat.inactive_count)/gig)
		log.Infof("darwin mem active: %.3f Gig", float64(ps*vmstat.active_count)/gig)
		log.Infof("darwin mem wired: %.3f Gig", float64(ps*vmstat.wire_count)/gig)
		log.Infof("darwin mem free: %.1f Meg", float64(ps*vmstat.free_count)/1048576.0)
		log.Infof("darwin mem zero filled: %.3f Gig", float64(ps*vmstat.zero_fill_count)/gig)
		log.Infof("darwin mem reactivated: %.3f Gig", float64(ps*vmstat.reactivations)/gig)
		log.Infof("darwin mem purgable: %.3f Gig", float64(ps*vmstat.purgeable_count)/gig)
	*/
	log.Debugf("darwin mem available: %.3f Gig", float64(ps*available)/gig)
	return map[string]float64{
		"Active_bytes":            float64(ps * vmstat.active_count),
		"Inactive_bytes":          float64(ps * vmstat.inactive_count),
		"wired_bytes_total":       float64(ps * vmstat.wire_count),
		"MemFree_bytes":           float64(ps * vmstat.free_count),
		"swapped_in_pages_total":  float64(ps * vmstat.pageins),
		"swapped_out_pages_total": float64(ps * vmstat.pageouts),
		"MemTotal_bytes":          float64(total),
		"MemAvailable_bytes":      float64(ps * available),
	}, nil
}
