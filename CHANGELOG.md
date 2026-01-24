# Changelog

## [0.1.2](https://github.com/garrettladley/thoop/compare/v0.1.1...v0.1.2) (2026-01-24)


### Bug Fixes

* **recovery page:** sleep performance w/ % sign ([#106](https://github.com/garrettladley/thoop/issues/106)) ([818bfe4](https://github.com/garrettladley/thoop/commit/818bfe439202a1bd464c6c0e577f0fe9b34b04ea))

## [0.1.1](https://github.com/garrettladley/thoop/compare/v0.1.0...v0.1.1) (2026-01-24)


### Features

* /auth/refresh & clean up thoop cmd ([#53](https://github.com/garrettladley/thoop/issues/53)) ([062be4f](https://github.com/garrettladley/thoop/commit/062be4f5d9da4c8a7049715e083bfebd46887cd9))
* `gauge.New` ([#25](https://github.com/garrettladley/thoop/issues/25)) ([c92d9b0](https://github.com/garrettladley/thoop/commit/c92d9b04f5e73980ef10e1c569abd33f16dff79f))
* `xslog.Error` and go_json ([#24](https://github.com/garrettladley/thoop/issues/24)) ([cc2a0a4](https://github.com/garrettladley/thoop/commit/cc2a0a41724d436b0fff713bdd4e0b4157c45b37))
* add version and clean color scheme to CLI ([#63](https://github.com/garrettladley/thoop/issues/63)) ([10b09b7](https://github.com/garrettladley/thoop/commit/10b09b775f21a042d3991240f4a57b9c3bb982fd))
* add workflow_dispatch to release-please ([#62](https://github.com/garrettladley/thoop/issues/62)) ([356a1f9](https://github.com/garrettladley/thoop/commit/356a1f94f7951c604ed7c93eafe55814f94578e0))
* api key auth ([#43](https://github.com/garrettladley/thoop/issues/43)) ([06ecad3](https://github.com/garrettladley/thoop/commit/06ecad3aa39f2ac599970fbfceaf1f8425304ac2))
* **auth/refresh:** slog real error on failed refresh ([#92](https://github.com/garrettladley/thoop/issues/92)) ([54da1be](https://github.com/garrettladley/thoop/commit/54da1be15e2cfc1931356f765aba8a784f3cdd55))
* **auth:** set oaut2.Endpoint AuthStyle ([#94](https://github.com/garrettladley/thoop/issues/94)) ([f2f5c95](https://github.com/garrettladley/thoop/commit/f2f5c95f2ce4a79e630a6690b3516a66ce888b3d))
* better ctx handling on shutdown ([#37](https://github.com/garrettladley/thoop/issues/37)) ([db28de1](https://github.com/garrettladley/thoop/commit/db28de1746740cba4d349829e8cdce50cab906dc))
* **cli:** major==0 always incompatible & tui prompt update ([#84](https://github.com/garrettladley/thoop/issues/84)) ([e5c0ad6](https://github.com/garrettladley/thoop/commit/e5c0ad60059178192a264573d47432e116746c6e))
* **cmds:** fire off dependent cmds in batch ([#18](https://github.com/garrettladley/thoop/issues/18)) ([ff7ea87](https://github.com/garrettladley/thoop/commit/ff7ea875a2d641ba47f4a2dd7c51e8bf40769971))
* codecov.yml ([#56](https://github.com/garrettladley/thoop/issues/56)) ([17c5a6d](https://github.com/garrettladley/thoop/commit/17c5a6d10010b3821fcc43f58e3800f093d6b1df))
* ctx as a dep ([#19](https://github.com/garrettladley/thoop/issues/19)) ([d6d5cab](https://github.com/garrettladley/thoop/commit/d6d5cab8983b556b7061b8b2a7cfc8bd319c5a76))
* ctx updates ([#29](https://github.com/garrettladley/thoop/issues/29)) ([211d30a](https://github.com/garrettladley/thoop/commit/211d30ab0379d5e22ec73072fd0eb75d1a46ce89))
* **ctx:** base ctx no sleep ([#48](https://github.com/garrettladley/thoop/issues/48)) ([c350b6b](https://github.com/garrettladley/thoop/commit/c350b6b3f516091f848bc84866d3e1d3480b1e96))
* **db:** embed migrations ([#12](https://github.com/garrettladley/thoop/issues/12)) ([f6ecdda](https://github.com/garrettladley/thoop/commit/f6ecddad0a5dedf51acc10566f04b731f5f2ac9c))
* **docker:** smol ([#8](https://github.com/garrettladley/thoop/issues/8)) ([2c38761](https://github.com/garrettladley/thoop/commit/2c387610fa33806aabe6fd637e3290ce92351fc7))
* drill pages & QoL fixes ([#104](https://github.com/garrettladley/thoop/issues/104)) ([00d83e0](https://github.com/garrettladley/thoop/commit/00d83e0fd4268f6a7633ef8472b70073d236598c))
* **dx:** .config/thoop-dev when deving ([#46](https://github.com/garrettladley/thoop/issues/46)) ([e03c526](https://github.com/garrettladley/thoop/commit/e03c526b0f8e37df39a49e38abffac8bc02dc4c2))
* **dx:** docker locally and fix parse whoop headers ([#22](https://github.com/garrettladley/thoop/issues/22)) ([7b05536](https://github.com/garrettladley/thoop/commit/7b05536b189f4bcbd5c0b3fdb47f0afb2e893d6f))
* **dx:** log level, version, dev footer ([#23](https://github.com/garrettladley/thoop/issues/23)) ([210cefa](https://github.com/garrettladley/thoop/commit/210cefa38ecdf9d49a7c9b4269c053b50febe40d))
* **dx:** run proxy flow locally ([#4](https://github.com/garrettladley/thoop/issues/4)) ([b89b154](https://github.com/garrettladley/thoop/commit/b89b1545f15add02650349eae156f76cc9f31629))
* **err:** err wrap ([#54](https://github.com/garrettladley/thoop/issues/54)) ([34b9551](https://github.com/garrettladley/thoop/commit/34b95514817d695d50c466a80efc6fdb50d78102))
* errgroup & log level ([#47](https://github.com/garrettladley/thoop/issues/47)) ([1ff231a](https://github.com/garrettladley/thoop/commit/1ff231a1e2bb13592d40e42823aba66058920cd8))
* flip build tags to tag dev instead of tag release ([#69](https://github.com/garrettladley/thoop/issues/69)) ([b798ef1](https://github.com/garrettladley/thoop/commit/b798ef1a34f28fbbb4168b08ab1a79421bbc5b40))
* fly proxy ([#3](https://github.com/garrettladley/thoop/issues/3)) ([2a70b3c](https://github.com/garrettladley/thoop/commit/2a70b3c503c12b06156c95fe3a0b2bdbb5e12d81))
* **fly:** `auto_stop_machines` -&gt; `suspend` ([#7](https://github.com/garrettladley/thoop/issues/7)) ([f565428](https://github.com/garrettladley/thoop/commit/f565428b7f399766dcb0fce0dda99c211c6c3c09))
* **fly:** only deploy on tag, not on main ([#71](https://github.com/garrettladley/thoop/issues/71)) ([bddfa82](https://github.com/garrettladley/thoop/commit/bddfa82b4a99748cd3993e8a322c8f22e3b4e972))
* gzip ([#51](https://github.com/garrettladley/thoop/issues/51)) ([8beec8f](https://github.com/garrettladley/thoop/commit/8beec8fc300758fbfb4e649c688fa34652a561e5))
* header lint ([#30](https://github.com/garrettladley/thoop/issues/30)) ([7ab4d8b](https://github.com/garrettladley/thoop/commit/7ab4d8b17fe1af2b2db50002b65f4eafb97b0ed5))
* initial release ([#61](https://github.com/garrettladley/thoop/issues/61)) ([7348631](https://github.com/garrettladley/thoop/commit/73486312691484eb1ba1ff288f4e9bc69338b5df))
* local cache, webhooks, sse ([#35](https://github.com/garrettladley/thoop/issues/35)) ([01df4c0](https://github.com/garrettladley/thoop/commit/01df4c096927e8f1fb7e160798662461b839d936))
* **logging:** revamp ([#33](https://github.com/garrettladley/thoop/issues/33)) ([d330093](https://github.com/garrettladley/thoop/commit/d3300931b5bebdeb5e51566928c1d1a01e02b6a0))
* **md:** rm TODO & create README ([#11](https://github.com/garrettladley/thoop/issues/11)) ([445d626](https://github.com/garrettladley/thoop/commit/445d62641e3958720099d00f2c0ad4d5f922cf8d))
* middleware pkg ([#31](https://github.com/garrettladley/thoop/issues/31)) ([dfb1e29](https://github.com/garrettladley/thoop/commit/dfb1e29c286095dbd5b69f83a77a933301944238))
* **notifs:** psql backing webhook storage & redis for notif ([#49](https://github.com/garrettladley/thoop/issues/49)) ([e0b0599](https://github.com/garrettladley/thoop/commit/e0b05994b7fc03faa27d9105c8fc638876dc91a5))
* oauth & sqlc scaffold ([#2](https://github.com/garrettladley/thoop/issues/2)) ([1c7ae00](https://github.com/garrettladley/thoop/commit/1c7ae004314bfa667a61194f70f8596b332973a8))
* **onboarding:** create user ([#41](https://github.com/garrettladley/thoop/issues/41)) ([6cfc9c1](https://github.com/garrettladley/thoop/commit/6cfc9c140be042d0a73b14ec0c7902b98b2e9ee0))
* **proxy:** hardening ([#5](https://github.com/garrettladley/thoop/issues/5)) ([ada9e8d](https://github.com/garrettladley/thoop/commit/ada9e8d466eeb924d5863c54cb962faa5391b2dd))
* **proxy:** proxy for requests to whoop api ([#21](https://github.com/garrettladley/thoop/issues/21)) ([5b36593](https://github.com/garrettladley/thoop/commit/5b3659352603855019a7d44fa7def4e9e323d4ae))
* redis backing ([#6](https://github.com/garrettladley/thoop/issues/6)) ([512e45c](https://github.com/garrettladley/thoop/commit/512e45c06a9ec79e505940509fb6b8eaa9576dcc))
* release on brew and go install ([#59](https://github.com/garrettladley/thoop/issues/59)) ([632c212](https://github.com/garrettladley/thoop/commit/632c212554130c2fd1075c12b59068c846469d70))
* scaffold ([#1](https://github.com/garrettladley/thoop/issues/1)) ([a0dbea7](https://github.com/garrettladley/thoop/commit/a0dbea7a4a3a4e6e9a9ba8b1b2f10596a752ea19))
* **semver:** revamp ([#101](https://github.com/garrettladley/thoop/issues/101)) ([6976d49](https://github.com/garrettladley/thoop/commit/6976d495671dfe692bbeed423220b4e5f5e7953e))
* **server:** psql init ([#39](https://github.com/garrettladley/thoop/issues/39)) ([58adf36](https://github.com/garrettladley/thoop/commit/58adf3600ee77b5074325fed17239c605137a7c7))
* **server:** revoke all api keys on reauth ([#97](https://github.com/garrettladley/thoop/issues/97)) ([0d28573](https://github.com/garrettladley/thoop/commit/0d285734ad0f4e9d9a0bc71641ebd4b7c8864df6))
* simplify xsync ([#58](https://github.com/garrettladley/thoop/issues/58)) ([70179b2](https://github.com/garrettladley/thoop/commit/70179b287ea8b7d7868e4a160fbc354a43a0b213))
* **splash:** skip on keypress ([#20](https://github.com/garrettladley/thoop/issues/20)) ([019de38](https://github.com/garrettladley/thoop/commit/019de384d181db3f73929014bda9c34ec1ade655))
* switch to pure Go sqlite driver ([#64](https://github.com/garrettladley/thoop/issues/64)) ([91704ae](https://github.com/garrettladley/thoop/commit/91704aeef82ac8137e69a5fe522fcd8d30c9d81e))
* **tests:** code cov ([#40](https://github.com/garrettladley/thoop/issues/40)) ([69d1d4c](https://github.com/garrettladley/thoop/commit/69d1d4ca95174cde0cc4d49b86c7e096d2289652))
* **tui:** auth flow ([#34](https://github.com/garrettladley/thoop/issues/34)) ([e8708d5](https://github.com/garrettladley/thoop/commit/e8708d501a463fd263842049b4e0742ee38b7246))
* **tui:** gauge init ([#17](https://github.com/garrettladley/thoop/issues/17)) ([827d80d](https://github.com/garrettladley/thoop/commit/827d80d478b836d821d8a4894ffccc4fc87c1916))
* **tui:** init ([#16](https://github.com/garrettladley/thoop/issues/16)) ([2be8bb0](https://github.com/garrettladley/thoop/commit/2be8bb05a47b694e76ea132899d202233578f699))
* **tui:** press any key to continue & local dx ([#90](https://github.com/garrettladley/thoop/issues/90)) ([7b63497](https://github.com/garrettladley/thoop/commit/7b6349729afbe1f0aefb184010a5d1cbb2632bee))
* **tui:** smoother gauge and tests ([#38](https://github.com/garrettladley/thoop/issues/38)) ([0c28a9e](https://github.com/garrettladley/thoop/commit/0c28a9e72d5b92b0b5ac1125b732bea50b374cdd))
* user agent ([#45](https://github.com/garrettladley/thoop/issues/45)) ([b8a3e10](https://github.com/garrettladley/thoop/commit/b8a3e1003145bd29796f724b58434a137751e059))
* **whoop:** client ([#15](https://github.com/garrettladley/thoop/issues/15)) ([670aa7c](https://github.com/garrettladley/thoop/commit/670aa7ca3747f0d2f0ab44e615d6f2a20f8b1454))
* xerrors pkg ([#50](https://github.com/garrettladley/thoop/issues/50)) ([fcfec08](https://github.com/garrettladley/thoop/commit/fcfec083401e5771f9e6561cf1091941603ffad5))


### Bug Fixes

* `RequestIDOption` ([#32](https://github.com/garrettladley/thoop/issues/32)) ([f8de05f](https://github.com/garrettladley/thoop/commit/f8de05fea56212ab82a4e9d651f001fc97328310))
* ack notifs on tui ([#55](https://github.com/garrettladley/thoop/issues/55)) ([ce9a407](https://github.com/garrettladley/thoop/commit/ce9a407b1509ffb62a1b611da3917f67bc76a9e2))
* **ci:** parse PR number from release-please output ([#74](https://github.com/garrettladley/thoop/issues/74)) ([cb8413b](https://github.com/garrettladley/thoop/commit/cb8413bae415002d50f50788be85c6688c2cda07))
* **cli:** call brew update on upgrade if is homebrew install ([#86](https://github.com/garrettladley/thoop/issues/86)) ([01dca71](https://github.com/garrettladley/thoop/commit/01dca71bda1a44eb62a3553f6936abb8128c70d1))
* **cli:** cask in upgrade cmd ([#82](https://github.com/garrettladley/thoop/issues/82)) ([bdffd23](https://github.com/garrettladley/thoop/commit/bdffd23e2f10274e09216385f2195160bea5ce41))
* **cli:** upgrade & unstable version cmp ([#88](https://github.com/garrettladley/thoop/issues/88)) ([0147d66](https://github.com/garrettladley/thoop/commit/0147d66143c4bd3eae408c71cfc868cf9b18b586))
* **fly:** ghwf to mark deployment as _ ([#9](https://github.com/garrettladley/thoop/issues/9)) ([4c1fa29](https://github.com/garrettladley/thoop/commit/4c1fa297d87a3913fe387805f342365d323e68e8))
* **go:** time.Duration over int with units suffix ([#10](https://github.com/garrettladley/thoop/issues/10)) ([dbf092f](https://github.com/garrettladley/thoop/commit/dbf092f07973b6dfa361631bfa81f79f03be7172))
* pass version to fly deploy build ([#76](https://github.com/garrettladley/thoop/issues/76)) ([5b5ca40](https://github.com/garrettladley/thoop/commit/5b5ca4051901a7a9fb89c62b9410b8144535a24e))
* release please ([#60](https://github.com/garrettladley/thoop/issues/60)) ([aa0bd58](https://github.com/garrettladley/thoop/commit/aa0bd58a164645a913eac853c5601cb013ad168b))
* release-please manifest config ([#65](https://github.com/garrettladley/thoop/issues/65)) ([886a688](https://github.com/garrettladley/thoop/commit/886a688cba27b24631950fca4d9f94ff38df0322))
* rm unused server pkg ([#13](https://github.com/garrettladley/thoop/issues/13)) ([c30452f](https://github.com/garrettladley/thoop/commit/c30452f6da4160c05cc02a7a3149ebfd6c968e9e))
* set api key in tui clients on auth complete ([#52](https://github.com/garrettladley/thoop/issues/52)) ([570e16e](https://github.com/garrettladley/thoop/commit/570e16e0830eec53ae53e9bdb25da22f7f5ebe08))
* sse api key & premature body close ([#57](https://github.com/garrettladley/thoop/issues/57)) ([eef2a10](https://github.com/garrettladley/thoop/commit/eef2a102f56229a2adeb9fa87f2efe2b1119504b))
* use PAT for release-please to trigger checks ([#67](https://github.com/garrettladley/thoop/issues/67)) ([8dbbaae](https://github.com/garrettladley/thoop/commit/8dbbaae95a8ce7c3e398b1c9d049661947294157))
* var _ satisfies interface checks ([#14](https://github.com/garrettladley/thoop/issues/14)) ([bb09b5e](https://github.com/garrettladley/thoop/commit/bb09b5e267fa803fbdd45937a1d6b7b237aa1ca7))

## [0.0.12](https://github.com/garrettladley/thoop/compare/v0.0.11...v0.0.12) (2026-01-08)


### Features

* **semver:** revamp ([#101](https://github.com/garrettladley/thoop/issues/101)) ([6976d49](https://github.com/garrettladley/thoop/commit/6976d495671dfe692bbeed423220b4e5f5e7953e))

## [0.0.11](https://github.com/garrettladley/thoop/compare/v0.0.10...v0.0.11) (2026-01-08)


### Features

* **server:** revoke all api keys on reauth ([#97](https://github.com/garrettladley/thoop/issues/97)) ([0d28573](https://github.com/garrettladley/thoop/commit/0d285734ad0f4e9d9a0bc71641ebd4b7c8864df6))

## [0.0.10](https://github.com/garrettladley/thoop/compare/v0.0.9...v0.0.10) (2026-01-07)


### Features

* **auth:** set oaut2.Endpoint AuthStyle ([#94](https://github.com/garrettladley/thoop/issues/94)) ([f2f5c95](https://github.com/garrettladley/thoop/commit/f2f5c95f2ce4a79e630a6690b3516a66ce888b3d))

## [0.0.9](https://github.com/garrettladley/thoop/compare/v0.0.8...v0.0.9) (2026-01-07)


### Features

* **auth/refresh:** slog real error on failed refresh ([#92](https://github.com/garrettladley/thoop/issues/92)) ([54da1be](https://github.com/garrettladley/thoop/commit/54da1be15e2cfc1931356f765aba8a784f3cdd55))

## [0.0.8](https://github.com/garrettladley/thoop/compare/v0.0.7...v0.0.8) (2026-01-05)


### Features

* **tui:** press any key to continue & local dx ([#90](https://github.com/garrettladley/thoop/issues/90)) ([7b63497](https://github.com/garrettladley/thoop/commit/7b6349729afbe1f0aefb184010a5d1cbb2632bee))

## [0.0.7](https://github.com/garrettladley/thoop/compare/v0.0.6...v0.0.7) (2026-01-05)


### Bug Fixes

* **cli:** upgrade & unstable version cmp ([#88](https://github.com/garrettladley/thoop/issues/88)) ([0147d66](https://github.com/garrettladley/thoop/commit/0147d66143c4bd3eae408c71cfc868cf9b18b586))

## [0.0.6](https://github.com/garrettladley/thoop/compare/v0.0.5...v0.0.6) (2026-01-05)


### Bug Fixes

* **cli:** call brew update on upgrade if is homebrew install ([#86](https://github.com/garrettladley/thoop/issues/86)) ([01dca71](https://github.com/garrettladley/thoop/commit/01dca71bda1a44eb62a3553f6936abb8128c70d1))

## [0.0.5](https://github.com/garrettladley/thoop/compare/v0.0.4...v0.0.5) (2026-01-05)


### Features

* **cli:** major==0 always incompatible & tui prompt update ([#84](https://github.com/garrettladley/thoop/issues/84)) ([e5c0ad6](https://github.com/garrettladley/thoop/commit/e5c0ad60059178192a264573d47432e116746c6e))

## [0.0.4](https://github.com/garrettladley/thoop/compare/v0.0.3...v0.0.4) (2026-01-04)


### Bug Fixes

* **cli:** cask in upgrade cmd ([#82](https://github.com/garrettladley/thoop/issues/82)) ([bdffd23](https://github.com/garrettladley/thoop/commit/bdffd23e2f10274e09216385f2195160bea5ce41))

## [0.0.3](https://github.com/garrettladley/thoop/compare/v0.0.2...v0.0.3) (2026-01-04)


### Bug Fixes

* pass version to fly deploy build ([#76](https://github.com/garrettladley/thoop/issues/76)) ([5b5ca40](https://github.com/garrettladley/thoop/commit/5b5ca4051901a7a9fb89c62b9410b8144535a24e))

## [0.0.2](https://github.com/garrettladley/thoop/compare/v0.0.1...v0.0.2) (2026-01-04)


### Features

* flip build tags to tag dev instead of tag release ([#69](https://github.com/garrettladley/thoop/issues/69)) ([b798ef1](https://github.com/garrettladley/thoop/commit/b798ef1a34f28fbbb4168b08ab1a79421bbc5b40))
* **fly:** only deploy on tag, not on main ([#71](https://github.com/garrettladley/thoop/issues/71)) ([bddfa82](https://github.com/garrettladley/thoop/commit/bddfa82b4a99748cd3993e8a322c8f22e3b4e972))


### Bug Fixes

* **ci:** parse PR number from release-please output ([#74](https://github.com/garrettladley/thoop/issues/74)) ([cb8413b](https://github.com/garrettladley/thoop/commit/cb8413bae415002d50f50788be85c6688c2cda07))

## [0.0.1](https://github.com/garrettladley/thoop/compare/v0.0.0...v0.0.1) (2026-01-04)


### Features

* switch to pure Go sqlite driver ([#64](https://github.com/garrettladley/thoop/issues/64)) ([91704ae](https://github.com/garrettladley/thoop/commit/91704aeef82ac8137e69a5fe522fcd8d30c9d81e))


### Bug Fixes

* release-please manifest config ([#65](https://github.com/garrettladley/thoop/issues/65)) ([886a688](https://github.com/garrettladley/thoop/commit/886a688cba27b24631950fca4d9f94ff38df0322))
* use PAT for release-please to trigger checks ([#67](https://github.com/garrettladley/thoop/issues/67)) ([8dbbaae](https://github.com/garrettladley/thoop/commit/8dbbaae95a8ce7c3e398b1c9d049661947294157))
