package contour

type TiledRasterProvider struct {
}

func (p *TiledRasterProvider) Next() Raster {
	return nil
}

func (p *TiledRasterProvider) HasNext() bool {
	return false
}
