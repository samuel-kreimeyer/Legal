package survey

// Point is a survey control or boundary point as recorded in a survey file.
type Point struct {
	// Number is the point identifier assigned in the survey software.
	// ARDOT point number ranges are defined in the Surveys Manual §1.6.
	Number int

	// Code is the ARDOT feature code describing the monument type.
	Code FeatureCode

	// Northing is the Y (north) coordinate in survey feet.
	Northing float64

	// Easting is the X (east) coordinate in survey feet.
	Easting float64

	// Elevation is the Z coordinate in survey feet. Zero when not recorded.
	Elevation float64

	// Description is the free-text note attached to the point in the survey file.
	Description string
}
