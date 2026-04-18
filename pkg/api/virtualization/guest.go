package virtualization

import (
	"net/url"

	"github.com/synology-community/go-synology/pkg/util"
)

type VDisk struct {
	ID         string `url:"-"           json:"vdisk_id,omitempty"`
	ImageName  string `url:"image_name"  json:"image_name,omitempty"`
	ImageID    string `url:"image_id"    json:"image_id,omitempty"`
	CreateType int64  `url:"create_type" json:"create_type"`
	Size       int64  `url:"vdisk_size"  json:"vdisk_size,omitempty"`
	Controller int64  `url:"-"           json:"controller,omitempty"`
	Unmap      bool   `url:"-"           json:"unmap,omitempty"`
}

type VNIC struct {
	ID     string `url:"network_id"      json:"network_id,omitempty"`
	Name   string `url:"network_name"    json:"network_name,omitempty"`
	Mac    string `url:"mac"             json:"mac,omitempty"`
	Model  int64  `url:"model,omitempty" json:"model,omitempty"`
	VnicID string `url:"model,omitempty" json:"vnic_id,omitempty"`
}

type (
	VDisks    []VDisk
	VNICs     []VNIC
	IsoImages []string
)

func (s IsoImages) EncodeValues(k string, v *url.Values) error {
	return util.EncodeValues(s, k, v)
}

func (s VDisks) EncodeValues(k string, v *url.Values) error {
	return util.EncodeValues(s, k, v)
}

func (s VNICs) EncodeValues(k string, v *url.Values) error {
	return util.EncodeValues(s, k, v)
}

type Guest struct {
	ID            string   `url:"guest_id,omitempty"        json:"guest_id"`
	Name          string   `url:"guest_name,omitempty"      json:"guest_name"`
	Description   string   `url:"description,omitempty"     json:"description"`
	Status        string   `url:"status,omitempty"          json:"status"`
	StorageID     string   `url:"storage_id,omitempty"      json:"storage_id"`
	StorageName   string   `url:"storage_name,omitempty"    json:"storage_name"`
	AutoRun       int64    `url:"autorun,omitempty"         json:"autorun"`
	VcpuNum       int64    `url:"vcpu_num,omitempty"        json:"vcpu_num"`
	VramSize      int64    `url:"vram_size,omitempty"       json:"vram_size"`
	Disks         VDisks   `url:"vdisks,omitempty"          json:"vdisks"`
	Networks      VNICs    `url:"vnics,omitempty"           json:"vnics"`
	IsoImages     []string `url:"iso_images,omitempty"      json:"iso_images,omitempty" form:"iso_images,omitempty"`
	AutoCleanTask bool     `url:"auto_clean_task,omitempty"`
}

type GuestList struct {
	Guests []Guest `url:"guests" json:"guests"`
}

type GetGuest struct {
	ID   string `form:"guest_id"   url:"guest_id"`
	Name string `form:"guest_name" url:"guest_name"`
}

type GuestUpdate struct {
	ID          string    `url:"guest_id"                 json:"guest_id"`
	Name        string    `url:"guest_name"               json:"guest_name"`
	NewName     string    `url:"new_guest_name,omitempty" json:"-"`
	Description string    `url:"description,omitempty"    json:"description"`
	IsoImages   IsoImages `url:"iso_images"               json:"iso_images"`
	AutoRun     int64     `url:"autorun"                  json:"autorun"`
	VcpuNum     int64     `url:"vcpu_num,omitempty"       json:"vcpu_num"`
	VramSize    int64     `url:"vram_size,omitempty"      json:"vram_size"`
}

type GuestUpdateResponse struct{}

// GuestSettings is the response shape of SYNO.Virtualization.Guest.get_setting v1.
// It is a superset of the short Guest struct and supplies the precursor values
// required to construct a diff-style payload for SYNO.Virtualization.Guest.set v1.
type GuestSettings struct {
	Autorun         int64                `json:"autorun"`
	BootFrom        string               `json:"boot_from"`
	CPUPassthru     bool                 `json:"cpu_passthru"`
	CPUPinNum       int64                `json:"cpu_pin_num"`
	CPUWeight       int64                `json:"cpu_weight"`
	Desc            string               `json:"desc"`
	HyperVEnlighten bool                 `json:"hyperv_enlighten"`
	IsGeneralVM    bool                  `json:"is_general_vm"`
	IsHAEnabled    bool                  `json:"is_ha_enabled"`
	IsoImages      []string              `json:"iso_images"`
	KbLayout       string                `json:"kb_layout"`
	MachineType    string                `json:"machine_type"`
	Name           string                `json:"name"`
	RepoID         string                `json:"repo_id"`
	SerialConsole  bool                  `json:"serial_console"`
	UsbVersion     int64                 `json:"usb_version"`
	Usbs           []string              `json:"usbs"`
	UseOvmf        bool                  `json:"use_ovmf"`
	VcpuNum        int64                 `json:"vcpu_num"`
	VDisks         []GuestSettingsVDisk  `json:"vdisks"`
	VideoCard      string                `json:"video_card"`
	VNICs          []GuestSettingsVNIC   `json:"vnics"`
	// VramSize is returned in KB (the set v1 input is in MB — divide by 1024).
	VramSize int64 `json:"vram_size"`
}

// GuestSettingsVDisk is the per-disk shape returned by get_setting.
type GuestSettingsVDisk struct {
	DevLimit       int64  `json:"dev_limit"`
	DevReservation int64  `json:"dev_reservation"`
	DevWeight      int64  `json:"dev_weight"`
	IopsEnable     bool   `json:"iops_enable"`
	IsDummy        bool   `json:"is_dummy"`
	IsMetaDisk     bool   `json:"is_meta_disk"`
	LunType        string `json:"lun_type"`
	Size           string `json:"size"` // bytes, as string
	Unmap          bool   `json:"unmap"`
	VdiskID        string `json:"vdisk_id"`
	VdiskMode      int64  `json:"vdisk_mode"`
}

// GuestSettingsVNIC is the per-NIC shape returned by get_setting.
type GuestSettingsVNIC struct {
	Mac         string `json:"mac"`
	NetworkID   string `json:"network_id"`
	PreferSriov bool   `json:"prefer_sriov"`
	VnicID      string `json:"vnic_id"`
	VnicType    int64  `json:"vnic_type"`
}

// GuestApplyRequest is the high-level input for GuestApply. The client reads
// current settings via get_setting, then builds an add/edit/del diff for set v1.
//
// Matching rule:
//   - Disks: an entry with a non-empty VdiskID matching an existing disk is an edit.
//     An entry with empty VdiskID is an add. Existing disks with no matching entry
//     are deleted.
//   - NICs: same, keyed on VnicID.
type GuestApplyRequest struct {
	ID          string
	UseOvmf     *bool     // nil = keep existing
	MachineType *string   // nil = keep existing
	VcpuNum     *int64    // nil = keep existing
	VramSizeMB  *int64    // nil = keep existing (value is MB)
	IsoImages   *[]string // nil = keep existing; empty slice = unmount all
	Disks       []DiskSpec
	NICs        []NICSpec
}

// DiskSpec is the desired state for a single vdisk.
type DiskSpec struct {
	VdiskID   string // empty → add
	Name      string // used for add; for edit, defaults to existing
	VdiskMode int64  // 16=SATA, 32=IDE, 64=VirtIO-SCSI
	Unmap     bool
	SizeGB    int64 // used for add (and edit-resize, not yet supported)
}

// NICSpec is the desired state for a single vnic.
type NICSpec struct {
	VnicID    string // empty → add
	NetworkID string
	Mac       string // required for add
	VnicType  int64  // 1=VirtIO, 2=E1000
}

// guestSetPayload is the full top-level body for SYNO.Virtualization.Guest.set v1.
// Fields not returned by get_setting are hardcoded to VMM-accepted defaults.
type guestSetPayload struct {
	GuestID              string                   `url:"guest_id"`
	Name                 string                   `url:"name"`
	VcpuNum              int64                    `url:"vcpu_num"`
	VramSize             int64                    `url:"vram_size"` // MB
	VramUnit             string                   `url:"vram_unit"`
	Bios                 string                   `url:"bios"`
	UseOvmf              bool                     `url:"use_ovmf"`
	OldUseOvmf           bool                     `url:"old_use_ovmf"`
	MachineType          string                   `url:"machine_type"`
	VideoCard            string                   `url:"video_card"`
	CPUWeight            int64                    `url:"cpu_weight"`
	CPUPassthru          bool                     `url:"cpu_passthru"`
	CPUPinNum            int64                    `url:"cpu_pin_num"`
	HypervEnlighten      bool                     `url:"hyperv_enlighten"`
	Desc                 string                   `url:"desc"`
	Autorun              int64                    `url:"autorun"`
	BootFrom             string                   `url:"boot_from"`
	SerialConsole        bool                     `url:"serial_console"`
	UsbVersion           int64                    `url:"usb_version"`
	Usbs                 jsonSlice                `url:"usbs"`
	IsoImages            jsonSlice                `url:"iso_images"`
	GuestPrivilege       jsonSlice                `url:"guest_privilege"`
	VdiskNum             int64                    `url:"vdisk_num"`
	VdisksAdd            jsonList[vdiskAddItem]   `url:"vdisks_add"`
	VdisksDel            jsonList[string]         `url:"vdisks_del"`
	VdisksEdit           jsonList[vdiskEditItem]  `url:"vdisks_edit"`
	VnicsAdd             jsonList[vnicAddItem]    `url:"vnics_add"`
	VnicsDel             jsonList[string]         `url:"vnics_del"`
	VnicsEdit            jsonList[vnicEditItem]   `url:"vnics_edit"`
	IncreaseAllocatedSize int64                   `url:"increaseAllocatedSize"`
	OrderChanged         bool                     `url:"order_changed"`
	EnoughMemory         bool                     `url:"enough_memory"`
	SynoVmmUiID          string                   `url:"synovmm_ui_id"`
}

// jsonSlice marshals a []string as a single JSON form value (e.g. ["a","b"]).
type jsonSlice []string

func (s jsonSlice) EncodeValues(k string, v *url.Values) error {
	return util.EncodeValues([]string(s), k, v)
}

// jsonList[T] marshals []T as a single JSON form value.
type jsonList[T any] []T

func (s jsonList[T]) EncodeValues(k string, v *url.Values) error {
	return util.EncodeValues([]T(s), k, v)
}

// vdiskAddItem is the diff record for adding a new disk.
type vdiskAddItem struct {
	Type            string `json:"type"` // "add"
	VdiskMode       int64  `json:"vdisk_mode"`
	Name            string `json:"name"`
	Unmap           bool   `json:"unmap"`
	IopsEnable      bool   `json:"iops_enable"`
	DevLimit        int64  `json:"dev_limit"`
	DevReservation  int64  `json:"dev_reservation"`
	DevWeight       int64  `json:"dev_weight"`
	SetByUser       bool   `json:"set_by_user"`
	VdiskSize       int64  `json:"vdisk_size"` // GB
	Idx             int    `json:"idx"`
	IsVdiskSizeEdit bool   `json:"is_vdisk_size_edit"`
	IsUnmapEdit     bool   `json:"is_unmap_edit"`
}

// vdiskEditItem is the diff record for editing an existing disk.
// NOTE: for disks, the "edit" marker is "old" (not "edit").
type vdiskEditItem struct {
	Type              string `json:"type"` // "old"
	VdiskID           string `json:"vdisk_id"`
	Name              string `json:"name"`
	VdiskMode         int64  `json:"vdisk_mode"`
	Unmap             bool   `json:"unmap"`
	Size              string `json:"size"` // bytes
	VdiskSize         int64  `json:"vdisk_size"` // GB
	LunType           string `json:"lun_type"`
	IsDummy           bool   `json:"is_dummy"`
	IsMetaDisk        bool   `json:"is_meta_disk"`
	SetByUser         bool   `json:"set_by_user"`
	Idx               int    `json:"idx"`
	OriIdx            int    `json:"ori_idx"`
	OriVdiskMode      int64  `json:"ori_vdisk_mode"`
	OriUnmap          bool   `json:"ori_unmap"`
	OriDevLimit       int64  `json:"ori_dev_limit"`
	OriDevReservation int64  `json:"ori_dev_reservation"`
	OriDevWeight      int64  `json:"ori_dev_weight"`
	OriIopsEnable     bool   `json:"ori_iops_enable"`
	DevLimit          int64  `json:"dev_limit"`
	DevReservation    int64  `json:"dev_reservation"`
	DevWeight         int64  `json:"dev_weight"`
	IopsEnable        bool   `json:"iops_enable"`
	IsVdiskSizeEdit   bool   `json:"is_vdisk_size_edit"`
	IsUnmapEdit       bool   `json:"is_unmap_edit"`
}

// vnicAddItem is the diff record for adding a new NIC.
type vnicAddItem struct {
	Type        string `json:"type"` // "add"
	VnicType    int64  `json:"vnic_type"`
	NetworkID   string `json:"network_id"`
	Mac         string `json:"mac"`
	PreferSriov bool   `json:"prefer_sriov"`
}

// vnicEditItem is the diff record for editing an existing NIC.
// NOTE: "ori_vnic_tpye" is a Synology API typo and must be preserved verbatim.
type vnicEditItem struct {
	Type           string `json:"type"` // "edit"
	VnicID         string `json:"vnic_id"`
	VnicType       int64  `json:"vnic_type"`
	NetworkID      string `json:"network_id"`
	PreferSriov    bool   `json:"prefer_sriov"`
	OriMac         string `json:"ori_mac"`
	OriPreferSriov bool   `json:"ori_prefer_sriov"`
	OriVnicType    int64  `json:"ori_vnic_tpye"` // sic: Synology API typo
}
