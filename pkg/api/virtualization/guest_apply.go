package virtualization

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/synology-community/go-synology/pkg/api"
	"github.com/synology-community/go-synology/pkg/api/virtualization/methods"
)

// getSettingRequest is the request body for get_setting.
type getSettingRequest struct {
	GuestID string `url:"guest_id"`
}

// GuestGetSetting fetches the full editable settings of a guest via
// SYNO.Virtualization.Guest.get_setting v1. Unlike GuestGet, this returns
// stable vdisk_id / vnic_id values and all ori_* precursors needed to
// build a diff-style payload for SYNO.Virtualization.Guest.set v1.
func (v *Client) GuestGetSetting(ctx context.Context, guestID string) (*GuestSettings, error) {
	return api.Post[GuestSettings](v.client, ctx, &getSettingRequest{GuestID: guestID}, methods.GuestGetSetting)
}

// GuestApply reads the guest's current settings, builds an add/edit/del diff
// against the desired DiskSpec/NICSpec slices, and submits the full set v1
// payload. UI-concern fields that are not returned by get_setting are populated
// with values VMM accepts (see guestSetPayload defaults below).
func (v *Client) GuestApply(ctx context.Context, req GuestApplyRequest) error {
	if req.ID == "" {
		return fmt.Errorf("GuestApply: request ID is required")
	}

	cur, err := v.GuestGetSetting(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("GuestApply: get_setting failed: %w", err)
	}

	useOvmf := cur.UseOvmf
	if req.UseOvmf != nil {
		useOvmf = *req.UseOvmf
	}
	machineType := cur.MachineType
	if req.MachineType != nil {
		machineType = *req.MachineType
	}
	vcpuNum := cur.VcpuNum
	if req.VcpuNum != nil {
		vcpuNum = *req.VcpuNum
	}
	// get_setting returns vram_size in KB; set v1 expects MB.
	vramSizeMB := cur.VramSize / 1024
	if req.VramSizeMB != nil {
		vramSizeMB = *req.VramSizeMB
	}
	isoImages := cur.IsoImages
	if req.IsoImages != nil {
		isoImages = *req.IsoImages
	}

	diskAdds, diskEdits, diskDels := buildDiskDiff(cur.VDisks, req.Disks)
	nicAdds, nicEdits, nicDels := buildNICDiff(cur.VNICs, req.NICs)

	uiID, err := newUUID()
	if err != nil {
		return fmt.Errorf("GuestApply: generate ui id: %w", err)
	}

	payload := guestSetPayload{
		GuestID:         req.ID,
		Name:            cur.Name,
		VcpuNum:         vcpuNum,
		VramSize:        vramSizeMB,
		VramUnit:        "1024",
		Bios:            biosFromOvmf(useOvmf),
		UseOvmf:         useOvmf,
		OldUseOvmf:      cur.UseOvmf,
		MachineType:     machineType,
		VideoCard:       cur.VideoCard,
		CPUWeight:       cur.CPUWeight,
		CPUPassthru:     cur.CPUPassthru,
		CPUPinNum:       cur.CPUPinNum,
		HypervEnlighten: cur.HyperVEnlighten,
		Desc:            cur.Desc,
		Autorun:         cur.Autorun,
		BootFrom:        cur.BootFrom,
		SerialConsole:   cur.SerialConsole,
		UsbVersion:      cur.UsbVersion,
		Usbs:            jsonSlice(cur.Usbs),
		IsoImages:       jsonSlice(isoImages),
		GuestPrivilege:  jsonSlice{},
		VdiskNum:        int64(len(cur.VDisks) - len(diskDels) + len(diskAdds)),
		VdisksAdd:       jsonList[vdiskAddItem](diskAdds),
		VdisksDel:       jsonList[string](diskDels),
		VdisksEdit:      jsonList[vdiskEditItem](diskEdits),
		VnicsAdd:        jsonList[vnicAddItem](nicAdds),
		VnicsDel:        jsonList[string](nicDels),
		VnicsEdit:       jsonList[vnicEditItem](nicEdits),
		EnoughMemory:    true,
		SynoVmmUiID:     uiID,
	}

	_, err = api.Post[api.Response](v.client, ctx, &payload, methods.GuestUpdate)
	if err != nil {
		return fmt.Errorf("GuestApply: set failed: %w", err)
	}
	return nil
}

func biosFromOvmf(useOvmf bool) string {
	if useOvmf {
		return "uefi"
	}
	return "legacy"
}

// buildDiskDiff matches desired DiskSpecs against existing disks by VdiskID.
// It returns (adds, edits, dels). Edits preserve the existing size and dev_*
// defaults from cur; only VdiskMode and Unmap are driven by the spec.
func buildDiskDiff(cur []GuestSettingsVDisk, desired []DiskSpec) ([]vdiskAddItem, []vdiskEditItem, []string) {
	byID := make(map[string]int, len(cur))
	for i, d := range cur {
		byID[d.VdiskID] = i
	}

	seen := make(map[string]bool, len(desired))
	var adds []vdiskAddItem
	var edits []vdiskEditItem

	nextIdx := len(cur)
	for i, spec := range desired {
		if spec.VdiskID == "" {
			adds = append(adds, vdiskAddItem{
				Type:           "add",
				VdiskMode:      spec.VdiskMode,
				Name:           spec.Name,
				Unmap:          spec.Unmap,
				IopsEnable:     false,
				DevLimit:       0,
				DevReservation: 0,
				DevWeight:      3,
				SetByUser:      true,
				VdiskSize:      spec.SizeGB,
				Idx:            nextIdx,
				IsVdiskSizeEdit: false,
				IsUnmapEdit:    false,
			})
			nextIdx++
			continue
		}
		idx, ok := byID[spec.VdiskID]
		if !ok {
			// Desired references an ID that no longer exists — treat as add.
			adds = append(adds, vdiskAddItem{
				Type:           "add",
				VdiskMode:      spec.VdiskMode,
				Name:           spec.Name,
				Unmap:          spec.Unmap,
				DevWeight:      3,
				SetByUser:      true,
				VdiskSize:      spec.SizeGB,
				Idx:            nextIdx,
			})
			nextIdx++
			continue
		}
		seen[spec.VdiskID] = true
		existing := cur[idx]
		name := spec.Name
		if name == "" {
			name = fmt.Sprintf("Virtual Disk %d", idx+1)
		}
		edits = append(edits, vdiskEditItem{
			Type:              "old",
			VdiskID:           existing.VdiskID,
			Name:              name,
			VdiskMode:         spec.VdiskMode,
			Unmap:             spec.Unmap,
			Size:              existing.Size,
			VdiskSize:         sizeBytesToGB(existing.Size),
			LunType:           existing.LunType,
			IsDummy:           existing.IsDummy,
			IsMetaDisk:        existing.IsMetaDisk,
			SetByUser:         true,
			Idx:               i,
			OriIdx:            idx,
			OriVdiskMode:      existing.VdiskMode,
			OriUnmap:          existing.Unmap,
			OriDevLimit:       existing.DevLimit,
			OriDevReservation: existing.DevReservation,
			OriDevWeight:      existing.DevWeight,
			OriIopsEnable:     existing.IopsEnable,
			DevLimit:          existing.DevLimit,
			DevReservation:    existing.DevReservation,
			DevWeight:         existing.DevWeight,
			IopsEnable:        existing.IopsEnable,
			IsVdiskSizeEdit:   false,
			IsUnmapEdit:       spec.Unmap != existing.Unmap,
		})
	}

	var dels []string
	for _, d := range cur {
		if !seen[d.VdiskID] {
			dels = append(dels, d.VdiskID)
		}
	}
	return adds, edits, dels
}

// buildNICDiff is the NIC counterpart to buildDiskDiff.
func buildNICDiff(cur []GuestSettingsVNIC, desired []NICSpec) ([]vnicAddItem, []vnicEditItem, []string) {
	byID := make(map[string]int, len(cur))
	for i, n := range cur {
		byID[n.VnicID] = i
	}
	seen := make(map[string]bool, len(desired))
	var adds []vnicAddItem
	var edits []vnicEditItem

	for _, spec := range desired {
		if spec.VnicID == "" {
			adds = append(adds, vnicAddItem{
				Type:        "add",
				VnicType:    spec.VnicType,
				NetworkID:   spec.NetworkID,
				Mac:         spec.Mac,
				PreferSriov: false,
			})
			continue
		}
		idx, ok := byID[spec.VnicID]
		if !ok {
			adds = append(adds, vnicAddItem{
				Type:      "add",
				VnicType:  spec.VnicType,
				NetworkID: spec.NetworkID,
				Mac:       spec.Mac,
			})
			continue
		}
		seen[spec.VnicID] = true
		existing := cur[idx]
		edits = append(edits, vnicEditItem{
			Type:           "edit",
			VnicID:         existing.VnicID,
			VnicType:       spec.VnicType,
			NetworkID:      spec.NetworkID,
			PreferSriov:    false,
			OriMac:         existing.Mac,
			OriPreferSriov: existing.PreferSriov,
			OriVnicType:    existing.VnicType,
		})
	}

	var dels []string
	for _, n := range cur {
		if !seen[n.VnicID] {
			dels = append(dels, n.VnicID)
		}
	}
	return adds, edits, dels
}

// sizeBytesToGB converts the string-encoded byte count returned by get_setting
// into the GB integer form expected by the set v1 vdisk_size field.
// Rounds down; VMM always reports exact GiB multiples for existing disks.
func sizeBytesToGB(s string) int64 {
	var b int64
	_, err := fmt.Sscanf(s, "%d", &b)
	if err != nil {
		return 0
	}
	return b / (1024 * 1024 * 1024)
}

// newUUID returns a random UUID v4 string. VMM's synovmm_ui_id field is a UI
// nonce — any valid UUID is accepted.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
