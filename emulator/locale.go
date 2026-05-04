package emulator

import (
	"math"
)

// Locale represents the QBP-native spatiotemporal coordinate.
// W: Time, XYZ: ECEF Unit Vector.
type Locale QWord

// NewLocaleFromLatLon converts human coordinates to a QBP Locale.
// Standard ECEF: X=Prime, Y=90E, Z=North.
func (c *CPU) NewLocaleFromLatLon(lat, lon, time float64) Locale {
	prec := c.GB.Precision()
	phi := lat * math.Pi / 180.0
	theta := lon * math.Pi / 180.0

	q := NewQWord(prec)
	q.W.SetFloat64(time)
	q.X.SetFloat64(math.Cos(phi) * math.Cos(theta))
	q.Y.SetFloat64(math.Cos(phi) * math.Sin(theta))
	q.Z.SetFloat64(math.Sin(phi))

	return Locale(q)
}

// ToJS converts the Locale vector to Three.js Y-Up coordinates.
// Go X (Prime) -> JS Z+
// Go Y (90E)   -> JS X+
// Go Z (North) -> JS Y+
func (l Locale) ToJS() (x, y, z float64) {
	jx, _ := l.Y.Float64() // Go Y -> JS X
	jy, _ := l.Z.Float64() // Go Z -> JS Y
	jz, _ := l.X.Float64() // Go X -> JS Z
	return jx, jy, jz
}

// ToLatLon converts a Locale back to geographic coordinates.
func (l Locale) ToLatLon() (lat, lon, time float64) {
	x, _ := l.X.Float64()
	y, _ := l.Y.Float64()
	z, _ := l.Z.Float64()
	t, _ := l.W.Float64()

	lat = math.Asin(z) * 180.0 / math.Pi
	lon = math.Atan2(y, x) * 180.0 / math.Pi
	return lat, lon, t
}
