get-sass:
	yarn

build: get-sass
	hugo

dev:
	hugo server --disableFastRender
