name='{{.Info.Name | tolower}}'
version='{{.Info.Version}}'
release='1'
desc='{{.Info.Summary}}'
homepage='{{.Info.Homepage}}'
maintainer='Example <user@example.com>'
architectures=('all')
license=('{{if .Info.License | ne ""}}{{.Info.License}}{{else}}custom:Unknown{{end}}')
provides=('{{.Info.Name | tolower}}')
conflicts=('{{.Info.Name | tolower}}')

deps=("python3")
deps_arch=("python")
deps_alpine=("python3")

build_deps=("python3" "python3-setuptools")
build_deps_arch=("python" "python-setuptools")
build_deps_alpine=("python3" "py3-setuptools")

sources=("https://files.pythonhosted.org/packages/source/{{.SourceURL.Filename | firstchar}}/{{.Info.Name}}/{{.SourceURL.Filename}}")
checksums=('blake2b-256:{{.SourceURL.Digests.blake2b_256}}')

build() {
	cd "$srcdir/{{.Info.Name}}-${version}"
	python3 setup.py build
}

package() {
	cd "$srcdir/{{.Info.Name}}-${version}"
	python3 setup.py install --root="${pkgdir}/" --optimize=1 || return 1
}
