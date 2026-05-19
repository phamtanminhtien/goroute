# Changelog

## [1.5.0](https://github.com/phamtanminhtien/goroute/compare/v1.4.1...v1.5.0) (2026-05-19)


### Features

* add support for input and output token pricing in model management and update docker configuration ([5186348](https://github.com/phamtanminhtien/goroute/commit/5186348003b83fb06750e1de3960fa60eecce982))
* implement custom provider management and registration support ([2c074b7](https://github.com/phamtanminhtien/goroute/commit/2c074b7bd0e004d131b027e68c592892807a4b1e))
* implement dynamic pricing updates in analytics service and update docker-compose port and volume mappings ([a91b2dd](https://github.com/phamtanminhtien/goroute/commit/a91b2ddeec2d145a6f295c2223d9e7afd22a2112))

## [1.4.1](https://github.com/phamtanminhtien/goroute/compare/v1.4.0...v1.4.1) (2026-05-19)


### Bug Fixes

* chat completion flow ([#45](https://github.com/phamtanminhtien/goroute/issues/45)) ([f318836](https://github.com/phamtanminhtien/goroute/commit/f318836132df757918b0673ecd1bea376e4f73f2))

## [1.4.0](https://github.com/phamtanminhtien/goroute/compare/v1.3.0...v1.4.0) (2026-05-19)


### Features

* add /v1/responses endpoint with support for OpenAI passthrough and Codex reconstruction ([f46baef](https://github.com/phamtanminhtien/goroute/commit/f46baefa82e524c4e9c3eb36b2f79ca5cf34f582))
* add AI request logging for chat completions ([9a105ea](https://github.com/phamtanminhtien/goroute/commit/9a105ea1920309ae22348fc67fa5e7a0ab3c9ea0))
* add Codex response mapping and stream transformation for chat completions ([67bbb9a](https://github.com/phamtanminhtien/goroute/commit/67bbb9a1b3736f3650109643b2a063c080ddc09e))
* add codex usage admin flow and token refresh ([016a2f6](https://github.com/phamtanminhtien/goroute/commit/016a2f643c4e7825c657fa531fa52dc07571fafb))
* add connection status toggling in codex usage ([854bd44](https://github.com/phamtanminhtien/goroute/commit/854bd4432833a37a5d622022f558870679bd5a45))
* add Docker support and implement automatic generation of default configuration files ([b4eed1d](https://github.com/phamtanminhtien/goroute/commit/b4eed1d20a57996194d2085c7ec1bad00a28f5e4))
* add enabled field to connections and model combos with toggle functionality in API and UI ([441aee3](https://github.com/phamtanminhtien/goroute/commit/441aee3323c32bbe1576bf7bd5655863636d6922))
* add Goroute logo asset and update sidebar branding to reflect renaming ([c3e78a8](https://github.com/phamtanminhtien/goroute/commit/c3e78a8d5c63e518d19638692c7e847242184da1))
* add motion-based entrance animations to Alert Dialog and Modal components ([d92afaa](https://github.com/phamtanminhtien/goroute/commit/d92afaa852d91c2aa4199bc6842cd5949fc36a72))
* add provider request mode and translated body support to flow logging for improved auditability ([9fa13ac](https://github.com/phamtanminhtien/goroute/commit/9fa13aca2ffdecf3b4c293c38972a704a5751fd0))
* add sm size variant to Button component ([8bd7523](https://github.com/phamtanminhtien/goroute/commit/8bd7523c7a7448c093c38613b4955361a0e7110d))
* add SQLite storage and refactor config records ([9021082](https://github.com/phamtanminhtien/goroute/commit/9021082b05587ae818c0f813f17b8d67fbd7175f))
* add streaming support for chat completions in core use case and HTTP transport ([be32a72](https://github.com/phamtanminhtien/goroute/commit/be32a72074d9cf881556486c436753b3a38412d5))
* add support for creating custom provider models via API and UI ([2fe0540](https://github.com/phamtanminhtien/goroute/commit/2fe0540ff8a04cf52f1c165b7d931d6c2518ca4a))
* **admin:** add provider CRUD endpoints ([#33](https://github.com/phamtanminhtien/goroute/issues/33)) ([133e593](https://github.com/phamtanminhtien/goroute/commit/133e5932b125fa4e531c916a6ed5bc7c015f039a))
* **auth:** protect admin routes with bearer auth ([e498f6b](https://github.com/phamtanminhtien/goroute/commit/e498f6b95935dd679421e15da672907b950b57c3))
* **chatcompletion:** add fallback policy and attempt logging ([#20](https://github.com/phamtanminhtien/goroute/issues/20)) ([7658ac3](https://github.com/phamtanminhtien/goroute/commit/7658ac3b327502237b119f098ce4b0e6e317a0e4))
* configure automated releases and Docker image publishing via release-please ([1432bee](https://github.com/phamtanminhtien/goroute/commit/1432bee3c6aeee9076faae77de6baa9b1ee00a64))
* enable RTK by default and skip recording if not applied ([d713a30](https://github.com/phamtanminhtien/goroute/commit/d713a302c0ac51254cf0d150636696bbaf829d46))
* enable RTK by default and update Docker volume mapping to a local directory ([2ea4bc7](https://github.com/phamtanminhtien/goroute/commit/2ea4bc7e9b5a4285d2617bd958e2dc54047b9d52))
* enable streaming for codex provider and update response handling ([dfbd0a9](https://github.com/phamtanminhtien/goroute/commit/dfbd0a94409c04c8b8a77e0883f5d76ad0c5a834))
* extract and store the last SSE event payload in flow logs for translated responses ([44da68a](https://github.com/phamtanminhtien/goroute/commit/44da68a01078f4926533e4908dc7ceb12b3b1ea3))
* implement AI request logs page with filtering, pagination, and detail view support ([63e22de](https://github.com/phamtanminhtien/goroute/commit/63e22de4e7676853acd5b709db844de5ee030c31))
* implement authentication system with login page, auth guard, and Vitest testing infrastructure ([1934c51](https://github.com/phamtanminhtien/goroute/commit/1934c51666ba94a77cfcc8cda9a2fcc618af24a3))
* implement automatic conversation session ID generation and input transformation for Codex provider requests ([26ac543](https://github.com/phamtanminhtien/goroute/commit/26ac543e98026cf7f54c5cc7da59012c8b83743a))
* implement backend analytics service and frontend API integration for usage metrics tracking ([b362401](https://github.com/phamtanminhtien/goroute/commit/b362401cc9e845718bea2d7ff61ddca351fa892c))
* implement connection runtime status tracking and display in the UI ([6f8920c](https://github.com/phamtanminhtien/goroute/commit/6f8920cc629e55aca8429b69d44ad09a5d4889ab))
* implement dynamic LLM logging configuration via new settings API and UI dashboard ([d54af40](https://github.com/phamtanminhtien/goroute/commit/d54af4065c9fe03682766fea963e3b1f34b65a5c))
* implement model combo management and update response streaming to hide provider-specific models ([a514e72](https://github.com/phamtanminhtien/goroute/commit/a514e725011c03eeceb20fc870e8e4f5309f2c70))
* implement OAuth flow for Codex provider including backend exchange and frontend integration ([437b42d](https://github.com/phamtanminhtien/goroute/commit/437b42dd73bc07d9842589fe2f4bb54baf58e97e))
* implement persistent request history tracking in sqlite storage and connection registry ([935bea7](https://github.com/phamtanminhtien/goroute/commit/935bea720276bf09a7c1fbe63f7e31bf9e19c90b))
* implement provider model testing and add app boot loader animation ([cd1f519](https://github.com/phamtanminhtien/goroute/commit/cd1f519ded521e83c20738896894c59c5d047c24))
* implement provider registry and upstream OpenAI client for chat completion routing ([#14](https://github.com/phamtanminhtien/goroute/issues/14)) ([a5cbf71](https://github.com/phamtanminhtien/goroute/commit/a5cbf714ea0e69c5f677e0db0c0c6d62f5a0a49f))
* implement provider registry architecture and add Codex and OpenAI support ([8f9400b](https://github.com/phamtanminhtien/goroute/commit/8f9400b50031864a3976e567449d08a73d4ecf4d))
* implement real-time server-side log streaming with SSE and add console log viewer page ([a457b72](https://github.com/phamtanminhtien/goroute/commit/a457b72d2e60668998f5e2dc7ed8efdd19d5cce8))
* implement shared component library with form and layout UI primitives ([c52be2e](https://github.com/phamtanminhtien/goroute/commit/c52be2eac48a0f7d822b6e7a36f41d440ac11467))
* implement system API key management and OpenAI-compatible authentication support ([4498ff0](https://github.com/phamtanminhtien/goroute/commit/4498ff071f8ec793d2de8518f18be16b3135950d))
* implement usage analytics dashboard with ECharts visualization and mock data integration ([0f536c1](https://github.com/phamtanminhtien/goroute/commit/0f536c1c1bbed83c0ad9d7353981553589e3bca9))
* initialize docs-aligned Go scaffold ([5ecfcd8](https://github.com/phamtanminhtien/goroute/commit/5ecfcd8c832cf249f42852d253b3a5563d447da5))
* initialize React frontend project with Vite, Tailwind CSS, and robust routing infrastructure ([15503b9](https://github.com/phamtanminhtien/goroute/commit/15503b942d6de14b979483cbae07f43f27cc03d0))
* introduce RTK request compression with configuration, storage, and API support ([80ec7ee](https://github.com/phamtanminhtien/goroute/commit/80ec7ee570b3bdc56d44edb96b62fdbba2aa5c73))
* migrate HTTP routing to go-chi/chi for improved path parameter handling ([bf4fc60](https://github.com/phamtanminhtien/goroute/commit/bf4fc60b7ec62ac4ba50f479bc89a810dd29606b))
* **observability:** add model metadata and request history ([#22](https://github.com/phamtanminhtien/goroute/issues/22)) ([a47c293](https://github.com/phamtanminhtien/goroute/commit/a47c2933c08e8a8b5c09b220995ae9a91301c4ca))
* **openai:** add streaming and common chat fields ([#21](https://github.com/phamtanminhtien/goroute/issues/21)) ([bf75a7c](https://github.com/phamtanminhtien/goroute/commit/bf75a7c638eaa0876615aaeb195a258320a2ae96))
* persist OAuth token_type and expires_in, improve console log colors ([6600a41](https://github.com/phamtanminhtien/goroute/commit/6600a411d22fabb366555ffc13c49889310f7bee))
* refactor Codex responses streaming pipeline ([ff26000](https://github.com/phamtanminhtien/goroute/commit/ff2600031dac7dbb9b0a9998e24aa150bba34ed7))
* refactor Codex usage page into a consolidated quota tracker with provider filtering ([b45dba0](https://github.com/phamtanminhtien/goroute/commit/b45dba0397cc64d6a39c5b95a37ca168c1734552))
* serve admin UI from configurable directory with SPA fallback support ([4f81004](https://github.com/phamtanminhtien/goroute/commit/4f810040bcaec0b575439216a8d5527dd8d94764))
* update chat completion schema to support multi-part content including images and implement image URL redaction for request logging ([d8b4697](https://github.com/phamtanminhtien/goroute/commit/d8b4697a3969f51ebde0406c1f164338701d2906))
* update model lists for codex and openai providers and add conditional connection validation to model testing UI ([19fe43e](https://github.com/phamtanminhtien/goroute/commit/19fe43eccdf1fa724df1efd6875c1febaf61465e))
* update UI quota display ([4a42dc2](https://github.com/phamtanminhtien/goroute/commit/4a42dc2514f9bf450ac5aa3e371f4ba8734e2b4a))
* **web:** add shared admin ui primitives ([#35](https://github.com/phamtanminhtien/goroute/issues/35)) ([706cc35](https://github.com/phamtanminhtien/goroute/commit/706cc357e1ef5103708fc8a0890d0c5a270c7c1c))
* **webui:** replace mock providers page with real API integration ([a6ee065](https://github.com/phamtanminhtien/goroute/commit/a6ee065dc8f4552b6f306532442a9f9cb511ebe0))


### Bug Fixes

* adjust padding on error and empty state alerts in usage analytics page ([f4044b4](https://github.com/phamtanminhtien/goroute/commit/f4044b417d6f187c94daed9e631f9244ae4c9d7c))
* **ci:** install pnpm before enabling pnpm cache ([#36](https://github.com/phamtanminhtien/goroute/issues/36)) ([b656020](https://github.com/phamtanminhtien/goroute/commit/b6560207a8313db166739ae24df519334c90fdf5))
* ensure text field is included in JSON marshaling for text-based content parts and update docker-compose volume mount ([021a63a](https://github.com/phamtanminhtien/goroute/commit/021a63ac2ac25b5ceaca58f0284db715a5657c28))
* persist default listen config ([#9](https://github.com/phamtanminhtien/goroute/issues/9)) ([548865d](https://github.com/phamtanminhtien/goroute/commit/548865d06f3a051bcfd813fa8b314c2491ef47fe))
* remove omitempty tag from instructions field to ensure it is always sent to codex provider ([347d5c7](https://github.com/phamtanminhtien/goroute/commit/347d5c765a5dd1f9ba1af808091ca130332d1ab4))
* **responses:** preserve completed payload shape ([8606354](https://github.com/phamtanminhtien/goroute/commit/8606354f5e04a169ae21fc42b30e6f3fbc926839))

## [1.3.0](https://github.com/phamtanminhtien/goroute/compare/v1.2.0...v1.3.0) (2026-05-18)


### Features

* extract and store the last SSE event payload in flow logs for translated responses ([44da68a](https://github.com/phamtanminhtien/goroute/commit/44da68a01078f4926533e4908dc7ceb12b3b1ea3))
* refactor Codex responses streaming pipeline ([ff26000](https://github.com/phamtanminhtien/goroute/commit/ff2600031dac7dbb9b0a9998e24aa150bba34ed7))

## [1.2.0](https://github.com/phamtanminhtien/goroute/compare/v1.1.0...v1.2.0) (2026-05-18)


### Features

* add /v1/responses endpoint with support for OpenAI passthrough and Codex reconstruction ([f46baef](https://github.com/phamtanminhtien/goroute/commit/f46baefa82e524c4e9c3eb36b2f79ca5cf34f582))
* add AI request logging for chat completions ([9a105ea](https://github.com/phamtanminhtien/goroute/commit/9a105ea1920309ae22348fc67fa5e7a0ab3c9ea0))
* add Codex response mapping and stream transformation for chat completions ([67bbb9a](https://github.com/phamtanminhtien/goroute/commit/67bbb9a1b3736f3650109643b2a063c080ddc09e))
* add codex usage admin flow and token refresh ([016a2f6](https://github.com/phamtanminhtien/goroute/commit/016a2f643c4e7825c657fa531fa52dc07571fafb))
* add connection status toggling in codex usage ([854bd44](https://github.com/phamtanminhtien/goroute/commit/854bd4432833a37a5d622022f558870679bd5a45))
* add Docker support and implement automatic generation of default configuration files ([b4eed1d](https://github.com/phamtanminhtien/goroute/commit/b4eed1d20a57996194d2085c7ec1bad00a28f5e4))
* add enabled field to connections and model combos with toggle functionality in API and UI ([441aee3](https://github.com/phamtanminhtien/goroute/commit/441aee3323c32bbe1576bf7bd5655863636d6922))
* add Goroute logo asset and update sidebar branding to reflect renaming ([c3e78a8](https://github.com/phamtanminhtien/goroute/commit/c3e78a8d5c63e518d19638692c7e847242184da1))
* add motion-based entrance animations to Alert Dialog and Modal components ([d92afaa](https://github.com/phamtanminhtien/goroute/commit/d92afaa852d91c2aa4199bc6842cd5949fc36a72))
* add provider request mode and translated body support to flow logging for improved auditability ([9fa13ac](https://github.com/phamtanminhtien/goroute/commit/9fa13aca2ffdecf3b4c293c38972a704a5751fd0))
* add sm size variant to Button component ([8bd7523](https://github.com/phamtanminhtien/goroute/commit/8bd7523c7a7448c093c38613b4955361a0e7110d))
* add SQLite storage and refactor config records ([9021082](https://github.com/phamtanminhtien/goroute/commit/9021082b05587ae818c0f813f17b8d67fbd7175f))
* add streaming support for chat completions in core use case and HTTP transport ([be32a72](https://github.com/phamtanminhtien/goroute/commit/be32a72074d9cf881556486c436753b3a38412d5))
* add support for creating custom provider models via API and UI ([2fe0540](https://github.com/phamtanminhtien/goroute/commit/2fe0540ff8a04cf52f1c165b7d931d6c2518ca4a))
* **admin:** add provider CRUD endpoints ([#33](https://github.com/phamtanminhtien/goroute/issues/33)) ([133e593](https://github.com/phamtanminhtien/goroute/commit/133e5932b125fa4e531c916a6ed5bc7c015f039a))
* **auth:** protect admin routes with bearer auth ([e498f6b](https://github.com/phamtanminhtien/goroute/commit/e498f6b95935dd679421e15da672907b950b57c3))
* **chatcompletion:** add fallback policy and attempt logging ([#20](https://github.com/phamtanminhtien/goroute/issues/20)) ([7658ac3](https://github.com/phamtanminhtien/goroute/commit/7658ac3b327502237b119f098ce4b0e6e317a0e4))
* configure automated releases and Docker image publishing via release-please ([1432bee](https://github.com/phamtanminhtien/goroute/commit/1432bee3c6aeee9076faae77de6baa9b1ee00a64))
* enable RTK by default and skip recording if not applied ([d713a30](https://github.com/phamtanminhtien/goroute/commit/d713a302c0ac51254cf0d150636696bbaf829d46))
* enable RTK by default and update Docker volume mapping to a local directory ([2ea4bc7](https://github.com/phamtanminhtien/goroute/commit/2ea4bc7e9b5a4285d2617bd958e2dc54047b9d52))
* enable streaming for codex provider and update response handling ([dfbd0a9](https://github.com/phamtanminhtien/goroute/commit/dfbd0a94409c04c8b8a77e0883f5d76ad0c5a834))
* implement AI request logs page with filtering, pagination, and detail view support ([63e22de](https://github.com/phamtanminhtien/goroute/commit/63e22de4e7676853acd5b709db844de5ee030c31))
* implement authentication system with login page, auth guard, and Vitest testing infrastructure ([1934c51](https://github.com/phamtanminhtien/goroute/commit/1934c51666ba94a77cfcc8cda9a2fcc618af24a3))
* implement backend analytics service and frontend API integration for usage metrics tracking ([b362401](https://github.com/phamtanminhtien/goroute/commit/b362401cc9e845718bea2d7ff61ddca351fa892c))
* implement connection runtime status tracking and display in the UI ([6f8920c](https://github.com/phamtanminhtien/goroute/commit/6f8920cc629e55aca8429b69d44ad09a5d4889ab))
* implement dynamic LLM logging configuration via new settings API and UI dashboard ([d54af40](https://github.com/phamtanminhtien/goroute/commit/d54af4065c9fe03682766fea963e3b1f34b65a5c))
* implement model combo management and update response streaming to hide provider-specific models ([a514e72](https://github.com/phamtanminhtien/goroute/commit/a514e725011c03eeceb20fc870e8e4f5309f2c70))
* implement OAuth flow for Codex provider including backend exchange and frontend integration ([437b42d](https://github.com/phamtanminhtien/goroute/commit/437b42dd73bc07d9842589fe2f4bb54baf58e97e))
* implement persistent request history tracking in sqlite storage and connection registry ([935bea7](https://github.com/phamtanminhtien/goroute/commit/935bea720276bf09a7c1fbe63f7e31bf9e19c90b))
* implement provider model testing and add app boot loader animation ([cd1f519](https://github.com/phamtanminhtien/goroute/commit/cd1f519ded521e83c20738896894c59c5d047c24))
* implement provider registry and upstream OpenAI client for chat completion routing ([#14](https://github.com/phamtanminhtien/goroute/issues/14)) ([a5cbf71](https://github.com/phamtanminhtien/goroute/commit/a5cbf714ea0e69c5f677e0db0c0c6d62f5a0a49f))
* implement provider registry architecture and add Codex and OpenAI support ([8f9400b](https://github.com/phamtanminhtien/goroute/commit/8f9400b50031864a3976e567449d08a73d4ecf4d))
* implement real-time server-side log streaming with SSE and add console log viewer page ([a457b72](https://github.com/phamtanminhtien/goroute/commit/a457b72d2e60668998f5e2dc7ed8efdd19d5cce8))
* implement shared component library with form and layout UI primitives ([c52be2e](https://github.com/phamtanminhtien/goroute/commit/c52be2eac48a0f7d822b6e7a36f41d440ac11467))
* implement system API key management and OpenAI-compatible authentication support ([4498ff0](https://github.com/phamtanminhtien/goroute/commit/4498ff071f8ec793d2de8518f18be16b3135950d))
* implement usage analytics dashboard with ECharts visualization and mock data integration ([0f536c1](https://github.com/phamtanminhtien/goroute/commit/0f536c1c1bbed83c0ad9d7353981553589e3bca9))
* initialize docs-aligned Go scaffold ([5ecfcd8](https://github.com/phamtanminhtien/goroute/commit/5ecfcd8c832cf249f42852d253b3a5563d447da5))
* initialize React frontend project with Vite, Tailwind CSS, and robust routing infrastructure ([15503b9](https://github.com/phamtanminhtien/goroute/commit/15503b942d6de14b979483cbae07f43f27cc03d0))
* introduce RTK request compression with configuration, storage, and API support ([80ec7ee](https://github.com/phamtanminhtien/goroute/commit/80ec7ee570b3bdc56d44edb96b62fdbba2aa5c73))
* migrate HTTP routing to go-chi/chi for improved path parameter handling ([bf4fc60](https://github.com/phamtanminhtien/goroute/commit/bf4fc60b7ec62ac4ba50f479bc89a810dd29606b))
* **observability:** add model metadata and request history ([#22](https://github.com/phamtanminhtien/goroute/issues/22)) ([a47c293](https://github.com/phamtanminhtien/goroute/commit/a47c2933c08e8a8b5c09b220995ae9a91301c4ca))
* **openai:** add streaming and common chat fields ([#21](https://github.com/phamtanminhtien/goroute/issues/21)) ([bf75a7c](https://github.com/phamtanminhtien/goroute/commit/bf75a7c638eaa0876615aaeb195a258320a2ae96))
* persist OAuth token_type and expires_in, improve console log colors ([6600a41](https://github.com/phamtanminhtien/goroute/commit/6600a411d22fabb366555ffc13c49889310f7bee))
* refactor Codex usage page into a consolidated quota tracker with provider filtering ([b45dba0](https://github.com/phamtanminhtien/goroute/commit/b45dba0397cc64d6a39c5b95a37ca168c1734552))
* serve admin UI from configurable directory with SPA fallback support ([4f81004](https://github.com/phamtanminhtien/goroute/commit/4f810040bcaec0b575439216a8d5527dd8d94764))
* update chat completion schema to support multi-part content including images and implement image URL redaction for request logging ([d8b4697](https://github.com/phamtanminhtien/goroute/commit/d8b4697a3969f51ebde0406c1f164338701d2906))
* update model lists for codex and openai providers and add conditional connection validation to model testing UI ([19fe43e](https://github.com/phamtanminhtien/goroute/commit/19fe43eccdf1fa724df1efd6875c1febaf61465e))
* update UI quota display ([4a42dc2](https://github.com/phamtanminhtien/goroute/commit/4a42dc2514f9bf450ac5aa3e371f4ba8734e2b4a))
* **web:** add shared admin ui primitives ([#35](https://github.com/phamtanminhtien/goroute/issues/35)) ([706cc35](https://github.com/phamtanminhtien/goroute/commit/706cc357e1ef5103708fc8a0890d0c5a270c7c1c))
* **webui:** replace mock providers page with real API integration ([a6ee065](https://github.com/phamtanminhtien/goroute/commit/a6ee065dc8f4552b6f306532442a9f9cb511ebe0))


### Bug Fixes

* adjust padding on error and empty state alerts in usage analytics page ([f4044b4](https://github.com/phamtanminhtien/goroute/commit/f4044b417d6f187c94daed9e631f9244ae4c9d7c))
* **ci:** install pnpm before enabling pnpm cache ([#36](https://github.com/phamtanminhtien/goroute/issues/36)) ([b656020](https://github.com/phamtanminhtien/goroute/commit/b6560207a8313db166739ae24df519334c90fdf5))
* ensure text field is included in JSON marshaling for text-based content parts and update docker-compose volume mount ([021a63a](https://github.com/phamtanminhtien/goroute/commit/021a63ac2ac25b5ceaca58f0284db715a5657c28))
* persist default listen config ([#9](https://github.com/phamtanminhtien/goroute/issues/9)) ([548865d](https://github.com/phamtanminhtien/goroute/commit/548865d06f3a051bcfd813fa8b314c2491ef47fe))
* remove omitempty tag from instructions field to ensure it is always sent to codex provider ([347d5c7](https://github.com/phamtanminhtien/goroute/commit/347d5c765a5dd1f9ba1af808091ca130332d1ab4))
* **responses:** preserve completed payload shape ([8606354](https://github.com/phamtanminhtien/goroute/commit/8606354f5e04a169ae21fc42b30e6f3fbc926839))

## [1.1.0](https://github.com/phamtanminhtien/goroute/compare/v1.0.0...v1.1.0) (2026-05-18)


### Features

* add connection status toggling in codex usage ([854bd44](https://github.com/phamtanminhtien/goroute/commit/854bd4432833a37a5d622022f558870679bd5a45))
* add enabled field to connections and model combos with toggle functionality in API and UI ([441aee3](https://github.com/phamtanminhtien/goroute/commit/441aee3323c32bbe1576bf7bd5655863636d6922))
* add sm size variant to Button component ([8bd7523](https://github.com/phamtanminhtien/goroute/commit/8bd7523c7a7448c093c38613b4955361a0e7110d))
* add support for creating custom provider models via API and UI ([2fe0540](https://github.com/phamtanminhtien/goroute/commit/2fe0540ff8a04cf52f1c165b7d931d6c2518ca4a))
* enable RTK by default and update Docker volume mapping to a local directory ([2ea4bc7](https://github.com/phamtanminhtien/goroute/commit/2ea4bc7e9b5a4285d2617bd958e2dc54047b9d52))
* implement AI request logs page with filtering, pagination, and detail view support ([63e22de](https://github.com/phamtanminhtien/goroute/commit/63e22de4e7676853acd5b709db844de5ee030c31))
* implement connection runtime status tracking and display in the UI ([6f8920c](https://github.com/phamtanminhtien/goroute/commit/6f8920cc629e55aca8429b69d44ad09a5d4889ab))
* implement model combo management and update response streaming to hide provider-specific models ([a514e72](https://github.com/phamtanminhtien/goroute/commit/a514e725011c03eeceb20fc870e8e4f5309f2c70))
* implement system API key management and OpenAI-compatible authentication support ([4498ff0](https://github.com/phamtanminhtien/goroute/commit/4498ff071f8ec793d2de8518f18be16b3135950d))
* update UI quota display ([4a42dc2](https://github.com/phamtanminhtien/goroute/commit/4a42dc2514f9bf450ac5aa3e371f4ba8734e2b4a))


### Bug Fixes

* ensure text field is included in JSON marshaling for text-based content parts and update docker-compose volume mount ([021a63a](https://github.com/phamtanminhtien/goroute/commit/021a63ac2ac25b5ceaca58f0284db715a5657c28))

## 1.0.0 (2026-05-18)


### Features

* add /v1/responses endpoint with support for OpenAI passthrough and Codex reconstruction ([f46baef](https://github.com/phamtanminhtien/goroute/commit/f46baefa82e524c4e9c3eb36b2f79ca5cf34f582))
* add AI request logging for chat completions ([9a105ea](https://github.com/phamtanminhtien/goroute/commit/9a105ea1920309ae22348fc67fa5e7a0ab3c9ea0))
* add Codex response mapping and stream transformation for chat completions ([67bbb9a](https://github.com/phamtanminhtien/goroute/commit/67bbb9a1b3736f3650109643b2a063c080ddc09e))
* add codex usage admin flow and token refresh ([016a2f6](https://github.com/phamtanminhtien/goroute/commit/016a2f643c4e7825c657fa531fa52dc07571fafb))
* add Docker support and implement automatic generation of default configuration files ([b4eed1d](https://github.com/phamtanminhtien/goroute/commit/b4eed1d20a57996194d2085c7ec1bad00a28f5e4))
* add Goroute logo asset and update sidebar branding to reflect renaming ([c3e78a8](https://github.com/phamtanminhtien/goroute/commit/c3e78a8d5c63e518d19638692c7e847242184da1))
* add motion-based entrance animations to Alert Dialog and Modal components ([d92afaa](https://github.com/phamtanminhtien/goroute/commit/d92afaa852d91c2aa4199bc6842cd5949fc36a72))
* add provider request mode and translated body support to flow logging for improved auditability ([9fa13ac](https://github.com/phamtanminhtien/goroute/commit/9fa13aca2ffdecf3b4c293c38972a704a5751fd0))
* add SQLite storage and refactor config records ([9021082](https://github.com/phamtanminhtien/goroute/commit/9021082b05587ae818c0f813f17b8d67fbd7175f))
* add streaming support for chat completions in core use case and HTTP transport ([be32a72](https://github.com/phamtanminhtien/goroute/commit/be32a72074d9cf881556486c436753b3a38412d5))
* **admin:** add provider CRUD endpoints ([#33](https://github.com/phamtanminhtien/goroute/issues/33)) ([133e593](https://github.com/phamtanminhtien/goroute/commit/133e5932b125fa4e531c916a6ed5bc7c015f039a))
* **auth:** protect admin routes with bearer auth ([e498f6b](https://github.com/phamtanminhtien/goroute/commit/e498f6b95935dd679421e15da672907b950b57c3))
* **chatcompletion:** add fallback policy and attempt logging ([#20](https://github.com/phamtanminhtien/goroute/issues/20)) ([7658ac3](https://github.com/phamtanminhtien/goroute/commit/7658ac3b327502237b119f098ce4b0e6e317a0e4))
* configure automated releases and Docker image publishing via release-please ([1432bee](https://github.com/phamtanminhtien/goroute/commit/1432bee3c6aeee9076faae77de6baa9b1ee00a64))
* enable RTK by default and skip recording if not applied ([d713a30](https://github.com/phamtanminhtien/goroute/commit/d713a302c0ac51254cf0d150636696bbaf829d46))
* enable streaming for codex provider and update response handling ([dfbd0a9](https://github.com/phamtanminhtien/goroute/commit/dfbd0a94409c04c8b8a77e0883f5d76ad0c5a834))
* implement authentication system with login page, auth guard, and Vitest testing infrastructure ([1934c51](https://github.com/phamtanminhtien/goroute/commit/1934c51666ba94a77cfcc8cda9a2fcc618af24a3))
* implement backend analytics service and frontend API integration for usage metrics tracking ([b362401](https://github.com/phamtanminhtien/goroute/commit/b362401cc9e845718bea2d7ff61ddca351fa892c))
* implement dynamic LLM logging configuration via new settings API and UI dashboard ([d54af40](https://github.com/phamtanminhtien/goroute/commit/d54af4065c9fe03682766fea963e3b1f34b65a5c))
* implement OAuth flow for Codex provider including backend exchange and frontend integration ([437b42d](https://github.com/phamtanminhtien/goroute/commit/437b42dd73bc07d9842589fe2f4bb54baf58e97e))
* implement persistent request history tracking in sqlite storage and connection registry ([935bea7](https://github.com/phamtanminhtien/goroute/commit/935bea720276bf09a7c1fbe63f7e31bf9e19c90b))
* implement provider model testing and add app boot loader animation ([cd1f519](https://github.com/phamtanminhtien/goroute/commit/cd1f519ded521e83c20738896894c59c5d047c24))
* implement provider registry and upstream OpenAI client for chat completion routing ([#14](https://github.com/phamtanminhtien/goroute/issues/14)) ([a5cbf71](https://github.com/phamtanminhtien/goroute/commit/a5cbf714ea0e69c5f677e0db0c0c6d62f5a0a49f))
* implement provider registry architecture and add Codex and OpenAI support ([8f9400b](https://github.com/phamtanminhtien/goroute/commit/8f9400b50031864a3976e567449d08a73d4ecf4d))
* implement real-time server-side log streaming with SSE and add console log viewer page ([a457b72](https://github.com/phamtanminhtien/goroute/commit/a457b72d2e60668998f5e2dc7ed8efdd19d5cce8))
* implement shared component library with form and layout UI primitives ([c52be2e](https://github.com/phamtanminhtien/goroute/commit/c52be2eac48a0f7d822b6e7a36f41d440ac11467))
* implement usage analytics dashboard with ECharts visualization and mock data integration ([0f536c1](https://github.com/phamtanminhtien/goroute/commit/0f536c1c1bbed83c0ad9d7353981553589e3bca9))
* initialize docs-aligned Go scaffold ([5ecfcd8](https://github.com/phamtanminhtien/goroute/commit/5ecfcd8c832cf249f42852d253b3a5563d447da5))
* initialize React frontend project with Vite, Tailwind CSS, and robust routing infrastructure ([15503b9](https://github.com/phamtanminhtien/goroute/commit/15503b942d6de14b979483cbae07f43f27cc03d0))
* introduce RTK request compression with configuration, storage, and API support ([80ec7ee](https://github.com/phamtanminhtien/goroute/commit/80ec7ee570b3bdc56d44edb96b62fdbba2aa5c73))
* migrate HTTP routing to go-chi/chi for improved path parameter handling ([bf4fc60](https://github.com/phamtanminhtien/goroute/commit/bf4fc60b7ec62ac4ba50f479bc89a810dd29606b))
* **observability:** add model metadata and request history ([#22](https://github.com/phamtanminhtien/goroute/issues/22)) ([a47c293](https://github.com/phamtanminhtien/goroute/commit/a47c2933c08e8a8b5c09b220995ae9a91301c4ca))
* **openai:** add streaming and common chat fields ([#21](https://github.com/phamtanminhtien/goroute/issues/21)) ([bf75a7c](https://github.com/phamtanminhtien/goroute/commit/bf75a7c638eaa0876615aaeb195a258320a2ae96))
* persist OAuth token_type and expires_in, improve console log colors ([6600a41](https://github.com/phamtanminhtien/goroute/commit/6600a411d22fabb366555ffc13c49889310f7bee))
* refactor Codex usage page into a consolidated quota tracker with provider filtering ([b45dba0](https://github.com/phamtanminhtien/goroute/commit/b45dba0397cc64d6a39c5b95a37ca168c1734552))
* serve admin UI from configurable directory with SPA fallback support ([4f81004](https://github.com/phamtanminhtien/goroute/commit/4f810040bcaec0b575439216a8d5527dd8d94764))
* update chat completion schema to support multi-part content including images and implement image URL redaction for request logging ([d8b4697](https://github.com/phamtanminhtien/goroute/commit/d8b4697a3969f51ebde0406c1f164338701d2906))
* update model lists for codex and openai providers and add conditional connection validation to model testing UI ([19fe43e](https://github.com/phamtanminhtien/goroute/commit/19fe43eccdf1fa724df1efd6875c1febaf61465e))
* **web:** add shared admin ui primitives ([#35](https://github.com/phamtanminhtien/goroute/issues/35)) ([706cc35](https://github.com/phamtanminhtien/goroute/commit/706cc357e1ef5103708fc8a0890d0c5a270c7c1c))
* **webui:** replace mock providers page with real API integration ([a6ee065](https://github.com/phamtanminhtien/goroute/commit/a6ee065dc8f4552b6f306532442a9f9cb511ebe0))


### Bug Fixes

* adjust padding on error and empty state alerts in usage analytics page ([f4044b4](https://github.com/phamtanminhtien/goroute/commit/f4044b417d6f187c94daed9e631f9244ae4c9d7c))
* **ci:** install pnpm before enabling pnpm cache ([#36](https://github.com/phamtanminhtien/goroute/issues/36)) ([b656020](https://github.com/phamtanminhtien/goroute/commit/b6560207a8313db166739ae24df519334c90fdf5))
* persist default listen config ([#9](https://github.com/phamtanminhtien/goroute/issues/9)) ([548865d](https://github.com/phamtanminhtien/goroute/commit/548865d06f3a051bcfd813fa8b314c2491ef47fe))
* remove omitempty tag from instructions field to ensure it is always sent to codex provider ([347d5c7](https://github.com/phamtanminhtien/goroute/commit/347d5c765a5dd1f9ba1af808091ca130332d1ab4))
