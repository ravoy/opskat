package rdp_svc

import (
	"testing"

	"github.com/octays/opskat/internal/model/entity/asset_entity"
	"github.com/stretchr/testify/require"
)

func TestClientOptionsEnableDesktopWallpaper(t *testing.T) {
	opts := clientOptions(&asset_entity.RDPConfig{}, "", 1280, 720)

	require.True(t, opts.Wallpaper)
}
