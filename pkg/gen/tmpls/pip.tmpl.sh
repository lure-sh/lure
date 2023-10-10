name='{{.name | tolower}}'
version='{{.version}}'
release='1'
desc='{{.description}}'
homepage='https://pypi.org/project/{{.name}}/'
maintainer='Example <user@example.com>'
architectures=('all')
license=('custom:Unknown')
provides=('{{.name | tolower}}')
conflicts=('{{.name | tolower}}')

deps=("python3")
deps_arch=("python")
deps_alpine=("python3")

build_deps=("python3" "python3-setuptools")
build_deps_arch=("python" "python-setuptools")
build_deps_alpine=("python3" "py3-setuptools")

sources=("https://files.pythonhosted.org/packages/source/{{.name | firstchar}}/{{.name}}/{{.name}}-${version}.tar.gz")
checksums=('{{.checksum}}')

build() {
	cd "$srcdir/{{.name}}-${version}"
	python3 setup.py build
}

package() {
	cd "$srcdir/{{.name}}-${version}"
	python3 setup.py install --root="${pkgdir}/" --optimize=1 || return 1
}
