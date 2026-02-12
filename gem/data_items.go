package gem

// DRACKCode enumerates S2F34 Define Report acknowledge codes.
type DRACKCode uint8

const (
	DRACKAccept          DRACKCode = 0
	DRACKInsufficient    DRACKCode = 1
	DRACKInvalidFormat   DRACKCode = 2
	DRACKRPTIDRedefined  DRACKCode = 3
	DRACKVIDDoesNotExist DRACKCode = 4
)

// LRACKCode enumerates S2F36 Link Event Report acknowledge codes.
type LRACKCode uint8

const (
	LRACKAccept            LRACKCode = 0
	LRACKInsufficient      LRACKCode = 1
	LRACKInvalidFormat     LRACKCode = 2
	LRACKCEIDAlreadyLinked LRACKCode = 3
	LRACKCEIDDoesNotExist  LRACKCode = 4
	LRACKRPTIDDoesNotExist LRACKCode = 5
)

// ERACKCode enumerates S2F38 Enable/Disable Event Report acknowledge codes.
type ERACKCode uint8

const (
	ERACKAccepted    ERACKCode = 0
	ERACKCEIDUnknown ERACKCode = 1
)

// ECACKCode enumerates S2F16 Equipment Constant acknowledge codes.
type ECACKCode uint8

const (
	ECACKAccepted        ECACKCode = 0
	ECACKDoesNotExist    ECACKCode = 1
	ECACKInvalidData     ECACKCode = 2
	ECACKValidationError ECACKCode = 3
)

func (c DRACKCode) Int() int { return int(c) }
func (c LRACKCode) Int() int { return int(c) }
func (c ERACKCode) Int() int { return int(c) }
func (c ECACKCode) Int() int { return int(c) }
