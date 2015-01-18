package main

type Site struct {
	CaskBase string
}

func NewSite(cask_base string) *Site {
	return &Site{CaskBase: cask_base}
}
