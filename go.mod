module gitlab.com/gitlab-org/cli

go 1.25.4

// NOTE: this is required because of the bug described in
// https://github.com/survivorbat/huhtest/issues/3
// The patch set for this replace is available here:
// https://github.com/timofurrer/huhtest/commit/46976a734937473c024ca7e50beee3c7a43fb528
replace github.com/survivorbat/huhtest => github.com/timofurrer/huhtest v0.0.0-20250922072747-46976a734937

require (
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/MakeNowJust/heredoc/v2 v2.0.1
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/adrg/xdg v0.5.3
	github.com/avast/retry-go/v4 v4.7.0
	github.com/briandowns/spinner v1.23.2
	github.com/charmbracelet/glamour v0.10.0
	github.com/charmbracelet/huh v0.8.0
	github.com/coder/websocket v1.8.14
	github.com/docker/cli v29.1.1+incompatible
	github.com/docker/docker-credential-helpers v0.9.4
	github.com/dustin/go-humanize v1.0.1
	github.com/gdamore/tcell/v2 v2.11.0
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/renameio/v2 v2.0.1
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.6.0
	github.com/gosuri/uilive v0.0.4
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-version v1.8.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/lunixbochs/vtclean v1.0.0
	github.com/mark3labs/mcp-go v0.43.1
	github.com/mattn/go-colorable v0.1.14
	github.com/mattn/go-isatty v0.0.20
	github.com/mattn/go-runewidth v0.0.19
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d
	github.com/mitchellh/go-homedir v1.1.0
	github.com/muesli/termenv v0.16.0
	github.com/otiai10/copy v1.14.1
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/pkg/errors v0.9.1
	github.com/rivo/tview v0.0.0-20230621164836-6cc0565babaf
	github.com/spf13/cobra v1.10.1
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/survivorbat/huhtest v0.0.2
	github.com/tidwall/pretty v1.2.1
	github.com/zalando/go-keyring v0.2.6
	gitlab.com/gitlab-org/api/client-go v1.0.1
	go.uber.org/goleak v1.3.0
	go.uber.org/mock v0.6.0
	golang.org/x/crypto v0.45.0
	golang.org/x/oauth2 v0.33.0
	golang.org/x/sync v0.18.0
	golang.org/x/term v0.37.0
	golang.org/x/text v0.31.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/apimachinery v0.34.2
	k8s.io/client-go v0.34.2
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397
	oss.terrastruct.com/d2 v0.7.1
)

require github.com/clipperhouse/uax29/v2 v2.2.0 // indirect

require (
	github.com/charmbracelet/lipgloss/v2 v2.0.0-beta.3.0.20250917201909-41ff0bf215ea
	github.com/charmbracelet/ultraviolet v0.0.0-20250915111650-81d4262876ef // indirect
	github.com/charmbracelet/x/exp/charmtone v0.0.0-20250603201427-c31516f43444 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/mango v0.1.0 // indirect
	github.com/muesli/mango-cobra v1.2.0 // indirect
	github.com/muesli/mango-pflag v0.1.0 // indirect
	github.com/muesli/roff v0.1.0 // indirect
)

require (
	al.essio.dev/pkg/shellescape v1.6.0 // indirect
	github.com/PuerkitoBio/goquery v1.10.0 // indirect
	github.com/alecthomas/chroma/v2 v2.14.0 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/catppuccin/go v0.3.0 // indirect
	github.com/charmbracelet/bubbles v0.21.1-0.20250623103423-23b8fd6302d7 // indirect
	github.com/charmbracelet/bubbletea v1.3.6 // indirect
	github.com/charmbracelet/colorprofile v0.3.2 // indirect
	github.com/charmbracelet/fang v0.4.3
	github.com/charmbracelet/lipgloss v1.1.1-0.20250404203927-76690c660834
	github.com/charmbracelet/x/ansi v0.10.1 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13 // indirect
	github.com/charmbracelet/x/exp/slice v0.0.0-20250327172914-2fdc97757edf // indirect
	github.com/charmbracelet/x/exp/strings v0.0.0-20240722160745-212f7b056ed0 // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/dop251/goja v0.0.0-20240927123429-241b342198c2 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/godbus/dbus/v5 v5.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/pprof v0.0.0-20241029153458-d1b30febd7db // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mazznoer/csscolorparser v0.1.5 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/otiai10/mint v1.6.3 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/yuin/goldmark v1.7.8 // indirect
	github.com/yuin/goldmark-emoji v1.0.5 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20240909161429-701f63a606c0 // indirect
	golang.org/x/image v0.20.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gotest.tools/v3 v3.5.2 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	oss.terrastruct.com/util-go v0.0.0-20250213174338-243d8661088a // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
