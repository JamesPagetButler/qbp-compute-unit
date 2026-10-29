package main

// The sigma* calibration (amend-1 item 3, option B): a unit-level pre-run with no
// dynamics, grid, probe or Stage 0. At each candidate lattice size (L = 16 and L = 24) and
// each sigma of {0.05, 0.10, 0.20, 0.40} it measures the perturbed null's pooled
// ambiguous-cell fraction (pi/2 rule) over 20 pinned fields, and applies the ruled rule.
// It touches only the calibration seeds 5850-5889 (tilt 5850-5869, field 5870-5889).

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

const (
	calTiltSeed0  = 5850
	calFieldSeed0 = 5870
	calFields     = 20
)

var calSizes = []int{16, 24}

func runCalibration(ensemble, out, pausePath string) int {
	tilt := make([]uint64, calFields)
	field := make([]uint64, calFields)
	for i := range tilt {
		tilt[i], field[i] = calTiltSeed0+uint64(i), calFieldSeed0+uint64(i)
	}
	for _, s := range append(append([]uint64{}, tilt...), field...) {
		if mdcu.KZClassifySeed(s) != mdcu.KZSeedTiltCal {
			fmt.Fprintf(os.Stderr, "internal error: seed %d is not a calibration seed\n", s)
			return 4
		}
	}
	f, err := os.Create(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	start := time.Now()
	emit := func(v any) {
		if err := enc.Encode(v); err != nil {
			w.Flush()
			fmt.Fprintln(os.Stderr, "encode:", err)
			os.Exit(6)
		}
	}
	emit(map[string]any{"kind": "meta", "mode": "calibration", "prereg_sha256": preregSHA, "ensemble": ensemble,
		"sizes": calSizes, "sigma_grid": mdcu.KZSigmaGrid, "ambig_max": mdcu.KZSigmaAmbigMax, "tilt_seeds": tilt,
		"field_seeds": field, "z2_rule": "amend-1: pi + edge pi/2", "go": runtime.Version(),
		"started": start.Format(time.RFC3339), "loadavg_start": loadavg()})
	p := mdcu.Amend1Z2Params()
	type star struct {
		L        int
		Sigma    float64
		Fallback bool
		Amp      float64
		AmpState string
	}
	var stars []star
	for _, L := range calSizes {
		if _, err := os.Stat(pausePath); err == nil {
			emit(map[string]any{"kind": "abort", "code": 3, "reason": "pause file present"})
			w.Flush()
			f.Close()
			fmt.Fprintln(os.Stderr, "abort: pause file present")
			return 3
		}
		L := L
		var rows []mdcu.KZCalRow
		var sc mdcu.KZSmoothCalibration
		if ensemble == "smooth" {
			sc = mdcu.KZCalibrateSmooth(L, field, tilt, p)
			rows = sc.Rows
			emit(map[string]any{"kind": "amplitude", "L": L, "A_half": sc.Amp.A, "untilted_pooled_ambig_frac": sc.Amp.Frac,
				"iterations": sc.Amp.Iterations, "status": sc.Amp.Status, "target": mdcu.KZAmpTargetFrac,
				"tolerance": mdcu.KZAmpTolerance, "bracket_hi": mdcu.KZAmpBracketHi})
		} else {
			mk := func(i int) [][4]float64 {
				if ensemble == "aligned" {
					return mdcu.NewKZS3FieldAligned(L).Q
				}
				return mdcu.NewKZS3FieldHaar(L, field[i]).Q
			}
			rows = mdcu.KZCalibrateSigma(L, mk, tilt, p)
		}
		for _, r := range rows {
			emit(map[string]any{"kind": "calibration", "L": L, "sigma": r.Sigma, "fields": r.Fields, "cells": r.Cells,
				"ambiguous": r.Ambiguous, "neg": r.NegCells, "defects": r.Defects,
				"pooled_ambig_frac": r.PooledAmbigFrac, "max_field_frac": r.MaxFieldFrac})
		}
		s, fb := mdcu.KZSigmaStar(rows)
		emit(map[string]any{"kind": "sigma_star", "L": L, "sigma_star": s, "fallback": fb})
		stars = append(stars, star{L, s, fb, sc.Amp.A, sc.Amp.Status})
	}
	emit(map[string]any{"kind": "end", "verdict": "calibration complete", "wall_s": time.Since(start).Seconds(), "loadavg": loadavg()})
	w.Flush()
	f.Close()
	fmt.Println("raw sha256:", fileSHA(out))
	for _, s := range stars {
		fmt.Printf("ensemble=%s  L=%d  A_half=%g (%s)  sigma*=%g  fallback=%v\n", ensemble, s.L, s.Amp, s.AmpState, s.Sigma, s.Fallback)
	}
	return 0
}
