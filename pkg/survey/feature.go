// Package survey provides types and utilities for reading ARDOT survey point
// files and correlating survey monumentation with parcel boundary geometry.
package survey

// FeatureCode is an ARDOT survey feature code as defined in the ARDOT Surveys
// Manual, Chapter 1, Section 1.7.
type FeatureCode string

// Feature codes relevant to land survey monumentation and boundary descriptions.
const (
	CodeIP  FeatureCode = "IP"  // Found Monument (iron pin, rebar, concrete)
	CodeFC  FeatureCode = "FC"  // Fence Corner
	CodeMBD FeatureCode = "MBD" // Standard Boundary Monument (set)
	CodeMRW FeatureCode = "MRW" // Standard Right-of-Way Monument (set)
	CodeMON FeatureCode = "MON" // Non-Standard Monument (set)
	CodeLC  FeatureCode = "LC"  // Calculated Corner (no physical monument)
	CodeRE  FeatureCode = "RE"  // Existing Right-of-Way point
	CodeXR  FeatureCode = "XR"  // Proposed Right-of-Way point
	CodePL  FeatureCode = "PL"  // Property Line
	CodeSD  FeatureCode = "SD"  // Subdivision Line
	CodeSC  FeatureCode = "SC"  // Section Line
	CodeQC  FeatureCode = "QC"  // Quarter Section Line
	CodeTC  FeatureCode = "TC"  // Sixteenth Section Line
	CodeBD  FeatureCode = "BD"  // Boundary
	CodeCOA FeatureCode = "COA" // Control of Access line
	CodeET  FeatureCode = "ET"  // Temporary Easement
	CodeML  FeatureCode = "ML"  // Meander Line
)

// MonumentKind classifies whether a survey point represents a physical
// object set or found in the field, a calculated position, or a ROW reference.
type MonumentKind int

const (
	// KindFound is a pre-existing monument located during fieldwork (e.g., IP).
	KindFound MonumentKind = iota
	// KindSet is a monument placed by the surveyor (e.g., MBD, MRW).
	KindSet
	// KindCalc is a computed position with no physical monument on the ground.
	KindCalc
	// KindROW is a point located on a right-of-way or easement line.
	KindROW
)

type featureInfo struct {
	Kind        MonumentKind
	Description string // phrase used after "being" in a legal description
}

// featureRegistry maps ARDOT feature codes to their legal-description phrases.
var featureRegistry = map[FeatureCode]featureInfo{
	CodeIP:  {KindFound, "a found iron pin"},
	CodeFC:  {KindFound, "a found fence corner"},
	CodeMBD: {KindSet, "a standard boundary monument set"},
	CodeMRW: {KindSet, "a standard right-of-way monument set"},
	CodeMON: {KindSet, "a non-standard monument set"},
	CodeLC:  {KindCalc, "a calculated point"},
	CodeRE:  {KindROW, "a point on the existing right-of-way"},
	CodeXR:  {KindROW, "a point on the proposed right-of-way"},
	CodeET:  {KindROW, "a point on the temporary easement boundary"},
}

// DescribeCode returns a human-readable phrase for the given feature code,
// suitable for use in a legal description after "being", and its MonumentKind.
// Unknown codes return a generic phrase and KindCalc.
func DescribeCode(code FeatureCode) (string, MonumentKind) {
	if info, ok := featureRegistry[code]; ok {
		return info.Description, info.Kind
	}
	return "a surveyed point", KindCalc
}
