// Copyright © 2017 Microsoft <wastore@microsoft.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package common

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"regexp"
	"sync/atomic"
	"time"

	"fmt"

	"os"

	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/azure-storage-file-go/azfile"
	"github.com/JeffreyRichter/enum/enum"
)

const (
	AZCOPY_PATH_SEPARATOR_STRING = "/"
	AZCOPY_PATH_SEPARATOR_CHAR   = '/'
	OS_PATH_SEPARATOR            = string(os.PathSeparator)
	Dev_Null                     = "/dev/null"

	//  this is the perm that AzCopy has used throughout its preview.  So, while we considered relaxing it to 0666
	//  we decided that the best option was to leave it as is, and only relax it if user feedback so requires.
	DEFAULT_FILE_PERM = 0644
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// this struct is used to parse the contents of file passed with list-of-files flag.
type ListOfFiles struct {
	Files []string
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type JobID UUID

func NewJobID() JobID {
	return JobID(NewUUID())
}

//var EmptyJobId JobID = JobID{}
func (j JobID) IsEmpty() bool {
	return j == JobID{}
}

func ParseJobID(jobID string) (JobID, error) {
	uuid, err := ParseUUID(jobID)
	if err != nil {
		return JobID{}, err
	}
	return JobID(uuid), nil
}

func (j JobID) String() string {
	return UUID(j).String()
}

// Implementing MarshalJSON() method for type JobID
func (j JobID) MarshalJSON() ([]byte, error) {
	return json.Marshal(UUID(j))
}

// Implementing UnmarshalJSON() method for type JobID
func (j *JobID) UnmarshalJSON(b []byte) error {
	var u UUID
	if err := json.Unmarshal(b, &u); err != nil {
		return err
	}
	*j = JobID(u)
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type PartNumber uint32
type Version uint32
type Status uint32

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type DeleteDestination uint32

var EDeleteDestination = DeleteDestination(0)

func (DeleteDestination) False() DeleteDestination  { return DeleteDestination(0) }
func (DeleteDestination) Prompt() DeleteDestination { return DeleteDestination(1) }
func (DeleteDestination) True() DeleteDestination   { return DeleteDestination(2) }

func (dd *DeleteDestination) Parse(s string) error {
	val, err := enum.Parse(reflect.TypeOf(dd), s, true)
	if err == nil {
		*dd = val.(DeleteDestination)
	}
	return err
}

func (dd DeleteDestination) String() string {
	return enum.StringInt(dd, reflect.TypeOf(dd))
}

type OutputFormat uint32

var EOutputFormat = OutputFormat(0)

func (OutputFormat) None() OutputFormat { return OutputFormat(0) }
func (OutputFormat) Text() OutputFormat { return OutputFormat(1) }
func (OutputFormat) Json() OutputFormat { return OutputFormat(2) }

func (of *OutputFormat) Parse(s string) error {
	val, err := enum.Parse(reflect.TypeOf(of), s, true)
	if err == nil {
		*of = val.(OutputFormat)
	}
	return err
}

func (of OutputFormat) String() string {
	return enum.StringInt(of, reflect.TypeOf(of))
}

var EExitCode = ExitCode(0)

type ExitCode uint32

func (ExitCode) Success() ExitCode { return ExitCode(0) }
func (ExitCode) Error() ExitCode   { return ExitCode(1) }

type LogLevel uint8

var ELogLevel = LogLevel(pipeline.LogNone)

func (LogLevel) None() LogLevel    { return LogLevel(pipeline.LogNone) }
func (LogLevel) Fatal() LogLevel   { return LogLevel(pipeline.LogFatal) }
func (LogLevel) Panic() LogLevel   { return LogLevel(pipeline.LogPanic) }
func (LogLevel) Error() LogLevel   { return LogLevel(pipeline.LogError) }
func (LogLevel) Warning() LogLevel { return LogLevel(pipeline.LogWarning) }
func (LogLevel) Info() LogLevel    { return LogLevel(pipeline.LogInfo) }
func (LogLevel) Debug() LogLevel   { return LogLevel(pipeline.LogDebug) }

func (ll *LogLevel) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(ll), s, true, true)
	if err == nil {
		*ll = val.(LogLevel)
	}
	return err
}

func (ll LogLevel) String() string {
	switch ll {
	case ELogLevel.None():
		return "NONE"
	case ELogLevel.Fatal():
		return "FATAL"
	case ELogLevel.Panic():
		return "PANIC"
	case ELogLevel.Error():
		return "ERR"
	case ELogLevel.Warning():
		return "WARN"
	case ELogLevel.Info():
		return "INFO"
	case ELogLevel.Debug():
		return "DBG"
	default:
		return enum.StringInt(ll, reflect.TypeOf(ll))
	}
}

func (ll LogLevel) ToPipelineLogLevel() pipeline.LogLevel {
	// This assumes that pipeline's LogLevel values can fit in a byte (which they can)
	return pipeline.LogLevel(ll)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var EJobPriority = JobPriority(0)

// JobPriority defines the transfer priorities supported by the Storage Transfer Engine's channels
// The default priority is Normal
type JobPriority uint8

func (JobPriority) Normal() JobPriority { return JobPriority(0) }
func (JobPriority) Low() JobPriority    { return JobPriority(1) }
func (jp JobPriority) String() string {
	return enum.StringInt(uint8(jp), reflect.TypeOf(jp))
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var EJobStatus = JobStatus(0)

// JobStatus indicates the status of a Job; the default is InProgress.
type JobStatus uint32 // Must be 32-bit for atomic operations

func (j *JobStatus) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(j), s, true, true)
	if err == nil {
		*j = val.(JobStatus)
	}
	return err
}

// Implementing MarshalJSON() method for type JobStatus
func (j JobStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.String())
}

// Implementing UnmarshalJSON() method for type JobStatus
func (j *JobStatus) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return j.Parse(s)
}

func (j *JobStatus) AtomicLoad() JobStatus {
	return JobStatus(atomic.LoadUint32((*uint32)(j)))
}

func (j *JobStatus) AtomicStore(newJobStatus JobStatus) {
	atomic.StoreUint32((*uint32)(j), uint32(newJobStatus))
}

func (j *JobStatus) EnhanceJobStatusInfo(skippedTransfers, failedTransfers, successfulTransfers bool) JobStatus {
	if failedTransfers && skippedTransfers {
		return EJobStatus.CompletedWithErrorsAndSkipped()
	} else if failedTransfers {
		if successfulTransfers {
			return EJobStatus.CompletedWithErrors()
		} else {
			return EJobStatus.Failed()
		}
	} else if skippedTransfers {
		return EJobStatus.CompletedWithSkipped()
	} else {
		return EJobStatus.Completed()
	}
}

func (j *JobStatus) IsJobDone() bool {
	return *j == EJobStatus.Completed() || *j == EJobStatus.Cancelled() || *j == EJobStatus.CompletedWithSkipped() ||
		*j == EJobStatus.CompletedWithErrors() || *j == EJobStatus.CompletedWithErrorsAndSkipped() ||
		*j == EJobStatus.Failed()
}

func (JobStatus) InProgress() JobStatus                    { return JobStatus(0) }
func (JobStatus) Paused() JobStatus                        { return JobStatus(1) }
func (JobStatus) Cancelling() JobStatus                    { return JobStatus(2) }
func (JobStatus) Cancelled() JobStatus                     { return JobStatus(3) }
func (JobStatus) Completed() JobStatus                     { return JobStatus(4) }
func (JobStatus) CompletedWithErrors() JobStatus           { return JobStatus(5) }
func (JobStatus) CompletedWithSkipped() JobStatus          { return JobStatus(6) }
func (JobStatus) CompletedWithErrorsAndSkipped() JobStatus { return JobStatus(7) }
func (JobStatus) Failed() JobStatus                        { return JobStatus(8) }
func (js JobStatus) String() string {
	return enum.StringInt(js, reflect.TypeOf(js))
}

////////////////////////////////////////////////////////////////

var ELocation = Location(0)

// Location indicates the type of Location
type Location uint8

func (Location) Unknown() Location { return Location(0) }
func (Location) Local() Location   { return Location(1) }
func (Location) Pipe() Location    { return Location(2) }
func (Location) Blob() Location    { return Location(3) }
func (Location) File() Location    { return Location(4) }
func (Location) BlobFS() Location  { return Location(5) }
func (Location) S3() Location      { return Location(6) }
func (l Location) String() string {
	return enum.StringInt(uint32(l), reflect.TypeOf(l))
}

// fromToValue returns the fromTo enum value for given
// from / To location combination. In 16 bits fromTo
// value, first 8 bits represents from location
func fromToValue(from Location, to Location) FromTo {
	return FromTo((FromTo(from) << 8) | FromTo(to))
}

func (l Location) IsRemote() bool {
	switch l {
	case ELocation.BlobFS(), ELocation.Blob(), ELocation.File(), ELocation.S3():
		return true
	case ELocation.Local(), ELocation.Pipe(), ELocation.Unknown():
		return false
	default:
		panic("unexpected location, please specify if it is remote")
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var EFromTo = FromTo(0)

// FromTo defines the different types of sources/destination location combinations
// FromTo is 16 bit where first 8 bit represents the from location and other 8 bits
// represents the to location
type FromTo uint16

func (FromTo) Unknown() FromTo   { return FromTo(0) }
func (FromTo) LocalBlob() FromTo { return FromTo(fromToValue(ELocation.Local(), ELocation.Blob())) }
func (FromTo) LocalFile() FromTo { return FromTo(fromToValue(ELocation.Local(), ELocation.File())) }
func (FromTo) BlobLocal() FromTo { return FromTo(fromToValue(ELocation.Blob(), ELocation.Local())) }
func (FromTo) FileLocal() FromTo { return FromTo(fromToValue(ELocation.File(), ELocation.Local())) }
func (FromTo) BlobPipe() FromTo  { return FromTo(fromToValue(ELocation.Blob(), ELocation.Pipe())) }
func (FromTo) PipeBlob() FromTo  { return FromTo(fromToValue(ELocation.Pipe(), ELocation.Blob())) }
func (FromTo) FilePipe() FromTo  { return FromTo(fromToValue(ELocation.File(), ELocation.Pipe())) }
func (FromTo) PipeFile() FromTo  { return FromTo(fromToValue(ELocation.Pipe(), ELocation.File())) }
func (FromTo) BlobTrash() FromTo { return FromTo(fromToValue(ELocation.Blob(), ELocation.Unknown())) }
func (FromTo) FileTrash() FromTo { return FromTo(fromToValue(ELocation.File(), ELocation.Unknown())) }
func (FromTo) BlobFSTrash() FromTo {
	return FromTo(fromToValue(ELocation.BlobFS(), ELocation.Unknown()))
}
func (FromTo) LocalBlobFS() FromTo { return FromTo(fromToValue(ELocation.Local(), ELocation.BlobFS())) }
func (FromTo) BlobFSLocal() FromTo { return FromTo(fromToValue(ELocation.BlobFS(), ELocation.Local())) }
func (FromTo) BlobBlob() FromTo    { return FromTo(fromToValue(ELocation.Blob(), ELocation.Blob())) }
func (FromTo) FileBlob() FromTo    { return FromTo(fromToValue(ELocation.File(), ELocation.Blob())) }
func (FromTo) S3Blob() FromTo      { return FromTo(fromToValue(ELocation.S3(), ELocation.Blob())) }

func (ft FromTo) String() string {
	return enum.StringInt(ft, reflect.TypeOf(ft))
}
func (ft *FromTo) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(ft), s, true, true)
	if err == nil {
		*ft = val.(FromTo)
	}
	return err
}

func (ft *FromTo) FromAndTo(s string) (srcLocation, dstLocation Location, err error) {
	srcLocation = ELocation.Unknown()
	dstLocation = ELocation.Unknown()
	val, err := enum.ParseInt(reflect.TypeOf(ft), s, true, true)
	if err == nil {
		dstLocation = Location(((1 << 8) - 1) & val.(FromTo))
		srcLocation = Location((((1 << 16) - 1) & val.(FromTo)) >> 8)
		return
	}
	err = fmt.Errorf("unable to parse the from and to Location from given FromTo %s", s)
	return
}

func (ft *FromTo) To() Location {
	return Location(((1 << 8) - 1) & *ft)
}

func (ft *FromTo) From() Location {
	return Location((((1 << 16) - 1) & *ft) >> 8)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Enumerates the values for blob type.
type BlobType uint8

var EBlobType = BlobType(0)

func (BlobType) None() BlobType { return BlobType(0) }

func (BlobType) BlockBlob() BlobType { return BlobType(1) }

func (BlobType) PageBlob() BlobType { return BlobType(2) }

func (BlobType) AppendBlob() BlobType { return BlobType(3) }

func (bt BlobType) String() string {
	return enum.StringInt(bt, reflect.TypeOf(bt))
}

func (bt *BlobType) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(bt), s, true, true)
	if err == nil {
		*bt = val.(BlobType)
	}
	return err
}

// ToAzBlobType returns the equivalent azblob.BlobType for given string.
func (bt *BlobType) ToAzBlobType() azblob.BlobType {
	blobType := bt.String()
	switch blobType {
	case string(azblob.BlobBlockBlob):
		return azblob.BlobBlockBlob
	case string(azblob.BlobPageBlob):
		return azblob.BlobPageBlob
	case string(azblob.BlobAppendBlob):
		return azblob.BlobAppendBlob
	default:
		return azblob.BlobNone
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var ETransferStatus = TransferStatus(0)

type TransferStatus int32 // Must be 32-bit for atomic operations; negative #s represent a specific failure code

// Transfer is ready to transfer and not started transferring yet
func (TransferStatus) NotStarted() TransferStatus { return TransferStatus(0) }

// TODO confirm whether this is actually needed
//   Outdated:
//     Transfer started & at least 1 chunk has successfully been transfered.
//     Used to resume a transfer that started to avoid transferring all chunks thereby improving performance
func (TransferStatus) Started() TransferStatus { return TransferStatus(1) }

// Transfer successfully completed
func (TransferStatus) Success() TransferStatus { return TransferStatus(2) }

// Transfer failed due to some error. This status does represent the state when transfer is cancelled
func (TransferStatus) Failed() TransferStatus { return TransferStatus(-1) }

// Transfer failed due to failure while Setting blob tier.
func (TransferStatus) BlobTierFailure() TransferStatus { return TransferStatus(-2) }

func (TransferStatus) BlobAlreadyExistsFailure() TransferStatus { return TransferStatus(-3) }

func (TransferStatus) FileAlreadyExistsFailure() TransferStatus { return TransferStatus(-4) }

func (TransferStatus) ADLSGen2PathAlreadyExistsFailure() TransferStatus { return TransferStatus(-5) }

func (ts TransferStatus) ShouldTransfer() bool {
	return ts == ETransferStatus.NotStarted() || ts == ETransferStatus.Started()
}
func (ts TransferStatus) DidFail() bool { return ts < 0 }

// Transfer is any of the three possible state (InProgress, Completer or Failed)
func (TransferStatus) All() TransferStatus { return TransferStatus(math.MaxInt8) }
func (ts TransferStatus) String() string {
	return enum.StringInt(ts, reflect.TypeOf(ts))
}
func (ts *TransferStatus) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(ts), s, false, true)
	if err == nil {
		*ts = val.(TransferStatus)
	}
	return err
}

// Implementing MarshalJSON() method for type Transfer Status
func (ts TransferStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(ts.String())
}

// Implementing UnmarshalJSON() method for type Transfer Status
func (ts *TransferStatus) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return ts.Parse(s)
}

func (ts *TransferStatus) AtomicLoad() TransferStatus {
	return TransferStatus(atomic.LoadInt32((*int32)(ts)))
}
func (ts *TransferStatus) AtomicStore(newTransferStatus TransferStatus) {
	atomic.StoreInt32((*int32)(ts), int32(newTransferStatus))
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var EBlockBlobTier = BlockBlobTier(0)

type BlockBlobTier uint8

func (BlockBlobTier) None() BlockBlobTier    { return BlockBlobTier(0) }
func (BlockBlobTier) Hot() BlockBlobTier     { return BlockBlobTier(1) }
func (BlockBlobTier) Cold() BlockBlobTier    { return BlockBlobTier(2) } // TODO: not sure why cold is here.
func (BlockBlobTier) Cool() BlockBlobTier    { return BlockBlobTier(3) }
func (BlockBlobTier) Archive() BlockBlobTier { return BlockBlobTier(4) }

func (bbt BlockBlobTier) String() string {
	return enum.StringInt(bbt, reflect.TypeOf(bbt))
}

func (bbt *BlockBlobTier) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(bbt), s, true, true)
	if err == nil {
		*bbt = val.(BlockBlobTier)
	}
	return err
}

func (bbt BlockBlobTier) ToAccessTierType() azblob.AccessTierType {
	return azblob.AccessTierType(bbt.String())
}

func (bbt BlockBlobTier) MarshalJSON() ([]byte, error) {
	return json.Marshal(bbt.String())
}

// Implementing UnmarshalJSON() method for type BlockBlobTier.
func (bbt *BlockBlobTier) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return bbt.Parse(s)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var EPageBlobTier = PageBlobTier(0)

type PageBlobTier uint8

func (PageBlobTier) None() PageBlobTier { return PageBlobTier(0) }
func (PageBlobTier) P10() PageBlobTier  { return PageBlobTier(10) }
func (PageBlobTier) P15() PageBlobTier  { return PageBlobTier(15) }
func (PageBlobTier) P20() PageBlobTier  { return PageBlobTier(20) }
func (PageBlobTier) P30() PageBlobTier  { return PageBlobTier(30) }
func (PageBlobTier) P4() PageBlobTier   { return PageBlobTier(4) }
func (PageBlobTier) P40() PageBlobTier  { return PageBlobTier(40) }
func (PageBlobTier) P50() PageBlobTier  { return PageBlobTier(50) }
func (PageBlobTier) P6() PageBlobTier   { return PageBlobTier(6) }

func (pbt PageBlobTier) String() string {
	return enum.StringInt(pbt, reflect.TypeOf(pbt))
}

func (pbt *PageBlobTier) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(pbt), s, true, true)
	if err == nil {
		*pbt = val.(PageBlobTier)
	}
	return err
}

func (pbt PageBlobTier) ToAccessTierType() azblob.AccessTierType {
	return azblob.AccessTierType(pbt.String())
}

func (pbt PageBlobTier) MarshalJSON() ([]byte, error) {
	return json.Marshal(pbt.String())
}

// Implementing UnmarshalJSON() method for type BlockBlobTier.
func (pbt *PageBlobTier) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return pbt.Parse(s)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var ECredentialType = CredentialType(0)

// CredentialType defines the different types of credentials
type CredentialType uint8

func (CredentialType) Unknown() CredentialType     { return CredentialType(0) }
func (CredentialType) OAuthToken() CredentialType  { return CredentialType(1) } // For Azure, OAuth
func (CredentialType) Anonymous() CredentialType   { return CredentialType(2) } // For Azure, SAS or public.
func (CredentialType) SharedKey() CredentialType   { return CredentialType(3) } // For Azure, SharedKey
func (CredentialType) S3AccessKey() CredentialType { return CredentialType(4) } // For S3, AccessKeyID and SecretAccessKey

func (ct CredentialType) String() string {
	return enum.StringInt(ct, reflect.TypeOf(ct))
}
func (ct *CredentialType) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(ct), s, true, true)
	if err == nil {
		*ct = val.(CredentialType)
	}
	return err
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var EHashValidationOption = HashValidationOption(0)

var DefaultHashValidationOption = EHashValidationOption.FailIfDifferent()

type HashValidationOption uint8

// FailIfDifferent says fail if hashes different, but NOT fail if saved hash is
// totally missing. This is a balance of convenience (for cases where no hash is saved) vs strictness
// (to validate strictly when one is present)
func (HashValidationOption) FailIfDifferent() HashValidationOption { return HashValidationOption(0) }

// Do not check hashes at download time at all
func (HashValidationOption) NoCheck() HashValidationOption { return HashValidationOption(1) }

// LogOnly means only log if missing or different, don't fail the transfer
func (HashValidationOption) LogOnly() HashValidationOption { return HashValidationOption(2) }

// FailIfDifferentOrMissing is the strictest option, and useful for testing or validation in cases when
// we _know_ there should be a hash
func (HashValidationOption) FailIfDifferentOrMissing() HashValidationOption {
	return HashValidationOption(3)
}

func (hvo HashValidationOption) String() string {
	return enum.StringInt(hvo, reflect.TypeOf(hvo))
}

func (hvo *HashValidationOption) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(hvo), s, true, true)
	if err == nil {
		*hvo = val.(HashValidationOption)
	}
	return err
}

func (hvo HashValidationOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(hvo.String())
}

func (hvo *HashValidationOption) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return hvo.Parse(s)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var EInvalidMetadataHandleOption = InvalidMetadataHandleOption(0)

var DefaultInvalidMetadataHandleOption = EInvalidMetadataHandleOption.ExcludeIfInvalid()

type InvalidMetadataHandleOption uint8

// ExcludeIfInvalid indicates whenever invalid metadata key is found, exclude the specific metadata with WARNING logged.
func (InvalidMetadataHandleOption) ExcludeIfInvalid() InvalidMetadataHandleOption {
	return InvalidMetadataHandleOption(0)
}

// FailIfInvalid indicates whenever invalid metadata key is found, directly fail the transfer.
func (InvalidMetadataHandleOption) FailIfInvalid() InvalidMetadataHandleOption {
	return InvalidMetadataHandleOption(1)
}

// RenameIfInvalid indicates whenever invalid metadata key is found, rename the metadata key and save the metadata with renamed key.
func (InvalidMetadataHandleOption) RenameIfInvalid() InvalidMetadataHandleOption {
	return InvalidMetadataHandleOption(2)
}

func (i InvalidMetadataHandleOption) String() string {
	return enum.StringInt(i, reflect.TypeOf(i))
}

func (i *InvalidMetadataHandleOption) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(i), s, true, true)
	if err == nil {
		*i = val.(InvalidMetadataHandleOption)
	}
	return err
}

func (i InvalidMetadataHandleOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *InvalidMetadataHandleOption) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return i.Parse(s)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
const (
	DefaultBlockBlobBlockSize = 8 * 1024 * 1024
	MaxBlockBlobBlockSize     = 100 * 1024 * 1024
	MaxAppendBlobBlockSize    = 4 * 1024 * 1024
	DefaultPageBlobChunkSize  = 4 * 1024 * 1024
	DefaultAzureFileChunkSize = 4 * 1024 * 1024
	MaxNumberOfBlocksPerBlob  = 50000
)

// This struct represent a single transfer entry with source and destination details
type CopyTransfer struct {
	Source           string
	Destination      string
	LastModifiedTime time.Time //represents the last modified time of source which ensures that source hasn't changed while transferring
	SourceSize       int64     // size of the source entity in bytes.

	// Properties for service to service copy
	ContentType        string
	ContentEncoding    string
	ContentDisposition string
	ContentLanguage    string
	CacheControl       string
	ContentMD5         []byte
	Metadata           Metadata

	// Properties for S2S blob copy
	BlobType azblob.BlobType
	BlobTier azblob.AccessTierType
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Metadata used in AzCopy.
type Metadata map[string]string

// ToAzBlobMetadata converts metadata to azblob's metadata.
func (m Metadata) ToAzBlobMetadata() azblob.Metadata {
	return azblob.Metadata(m)
}

// ToAzFileMetadata converts metadata to azfile's metadata.
func (m Metadata) ToAzFileMetadata() azfile.Metadata {
	return azfile.Metadata(m)
}

// FromAzBlobMetadataToCommonMetadata converts azblob's metadata to common metadata.
func FromAzBlobMetadataToCommonMetadata(m azblob.Metadata) Metadata {
	return Metadata(m)
}

// FromAzFileMetadataToCommonMetadata converts azfile's metadata to common metadata.
func FromAzFileMetadataToCommonMetadata(m azfile.Metadata) Metadata {
	return Metadata(m)
}

// Marshal marshals metadata to string.
func (m Metadata) Marshal() (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// UnMarshalToCommonMetadata unmarshals string to common metadata.
func UnMarshalToCommonMetadata(metadataString string) (Metadata, error) {
	var result Metadata
	if metadataString != "" {
		err := json.Unmarshal([]byte(metadataString), &result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// isValidMetadataKey checks if the given string is a valid metadata key for Azure.
// For Azure, metadata key must adhere to the naming rules for C# identifiers.
// As testing, reserved keyworkds for C# identifiers are also valid metadata key. (e.g. this, int)
// TODO: consider to use "[A-Za-z_]\w*" to replace this implementation, after ensuring the complexity is O(N).
func isValidMetadataKey(key string) bool {
	for i := 0; i < len(key); i++ {
		if i != 0 { // Most of case i != 0
			if !isValidMetadataKeyChar(key[i]) {
				return false
			}
			// Coming key is valid
		} else { // i == 0
			if !isValidMetadataKeyFirstChar(key[i]) {
				return false
			}
			// First key is valid
		}
	}

	return true
}

func isValidMetadataKeyChar(c byte) bool {
	if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_' {
		return true
	}
	return false
}

func isValidMetadataKeyFirstChar(c byte) bool {
	if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_' {
		return true
	}
	return false
}

func (m Metadata) ExcludeInvalidKey() (retainedMetadata Metadata, excludedMetadata Metadata, invalidKeyExists bool) {
	retainedMetadata = make(map[string]string)
	excludedMetadata = make(map[string]string)
	for k, v := range m {
		if isValidMetadataKey(k) {
			retainedMetadata[k] = v
		} else {
			invalidKeyExists = true
			excludedMetadata[k] = v
		}
	}

	return
}

const metadataRenamedKeyPrefix = "rename_"
const metadataKeyForRenamedOriginalKeyPrefix = "rename_key_"

var metadataKeyInvalidCharRegex = regexp.MustCompile("\\W")
var metadataKeyRenameErrStr = "failed to rename invalid metadata key %q"

// ResolveInvalidKey resolves invalid metadata key with following steps:
// 1. replace all invalid char(i.e. ASCII chars expect [0-9A-Za-z_]) with '_'
// 2. add 'rename_' as prefix for the new valid key, this key will be used to save original metadata's value.
// 3. add 'rename_key_' as prefix for the new valid key, this key will be used to save original metadata's invalid key.
// Example, given invalid metadata for Azure: '123-invalid':'content', it will be resolved as two new k:v pairs:
// 'rename_123_invalid':'content'
// 'rename_key_123_invalid':'123-invalid'
// So user can try to recover the metadata in Azure side.
// Note: To keep first version simple, whenever collision is found during key resolving, error will be returned.
// This can be further improved once any user feedback get.
func (m Metadata) ResolveInvalidKey() (resolvedMetadata Metadata, err error) {
	resolvedMetadata = make(map[string]string)

	hasCollision := func(name string) bool {
		_, hasCollisionToOrgNames := m[name]
		_, hasCollisionToNewNames := resolvedMetadata[name]

		return hasCollisionToOrgNames || hasCollisionToNewNames
	}

	for k, v := range m {
		if !isValidMetadataKey(k) {
			validKey := metadataKeyInvalidCharRegex.ReplaceAllString(k, "_")
			renamedKey := metadataRenamedKeyPrefix + validKey
			keyForRenamedOriginalKey := metadataKeyForRenamedOriginalKeyPrefix + validKey
			if hasCollision(renamedKey) || hasCollision(keyForRenamedOriginalKey) {
				return nil, fmt.Errorf(metadataKeyRenameErrStr, k)
			}

			resolvedMetadata[renamedKey] = v
			resolvedMetadata[keyForRenamedOriginalKey] = k
		} else {
			resolvedMetadata[k] = v
		}
	}

	return resolvedMetadata, nil
}

func (m Metadata) ConcatenatedKeys() string {
	buf := bytes.Buffer{}

	for k := range m {
		buf.WriteString("'")
		buf.WriteString(k)
		buf.WriteString("' ")
	}

	return buf.String()
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Common resource's HTTP headers stands for properties used in AzCopy.
type ResourceHTTPHeaders struct {
	ContentType        string
	ContentMD5         []byte
	ContentEncoding    string
	ContentLanguage    string
	ContentDisposition string
	CacheControl       string
}

// ToAzBlobHTTPHeaders converts ResourceHTTPHeaders to azblob's BlobHTTPHeaders.
func (h ResourceHTTPHeaders) ToAzBlobHTTPHeaders() azblob.BlobHTTPHeaders {
	return azblob.BlobHTTPHeaders{
		ContentType:        h.ContentType,
		ContentMD5:         h.ContentMD5,
		ContentEncoding:    h.ContentEncoding,
		ContentLanguage:    h.ContentLanguage,
		ContentDisposition: h.ContentDisposition,
		CacheControl:       h.CacheControl,
	}
}

// ToAzFileHTTPHeaders converts ResourceHTTPHeaders to azfile's FileHTTPHeaders.
func (h ResourceHTTPHeaders) ToAzFileHTTPHeaders() azfile.FileHTTPHeaders {
	return azfile.FileHTTPHeaders{
		ContentType:        h.ContentType,
		ContentMD5:         h.ContentMD5,
		ContentEncoding:    h.ContentEncoding,
		ContentLanguage:    h.ContentLanguage,
		ContentDisposition: h.ContentDisposition,
		CacheControl:       h.CacheControl,
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var ETransferDirection = TransferDirection(0)

type TransferDirection int32

func (TransferDirection) UnKnown() TransferDirection  { return TransferDirection(0) }
func (TransferDirection) Upload() TransferDirection   { return TransferDirection(1) }
func (TransferDirection) Download() TransferDirection { return TransferDirection(2) }
func (TransferDirection) S2SCopy() TransferDirection  { return TransferDirection(3) }

func (td TransferDirection) String() string {
	return enum.StringInt(td, reflect.TypeOf(td))
}
func (td *TransferDirection) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(td), s, false, true)
	if err == nil {
		*td = val.(TransferDirection)
	}
	return err
}

func (td *TransferDirection) AtomicLoad() TransferDirection {
	return TransferDirection(atomic.LoadInt32((*int32)(td)))
}
func (td *TransferDirection) AtomicStore(newTransferDirection TransferDirection) {
	atomic.StoreInt32((*int32)(td), int32(newTransferDirection))
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var EPerfConstraint = PerfConstraint(0)

type PerfConstraint int32

func (PerfConstraint) Unknown() PerfConstraint { return PerfConstraint(0) }
func (PerfConstraint) Disk() PerfConstraint    { return PerfConstraint(1) }
func (PerfConstraint) Service() PerfConstraint { return PerfConstraint(2) }

// others will be added in future

func (pc PerfConstraint) String() string {
	return enum.StringInt(pc, reflect.TypeOf(pc))
}

func (pc *PerfConstraint) Parse(s string) error {
	val, err := enum.ParseInt(reflect.TypeOf(pc), s, false, true)
	if err == nil {
		*pc = val.(PerfConstraint)
	}
	return err
}
