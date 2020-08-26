module github.com/GoogleCloudPlatform/cloud-run-release-manager

go 1.14

require (
	cloud.google.com/go v0.60.0
	cloud.google.com/go/pubsub v1.3.1
	github.com/TV4/logrus-stackdriver-formatter v0.1.0
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/jonboulle/clockwork v0.2.0
	github.com/mattn/go-isatty v0.0.12
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.28.0
	k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go v12.0.0+incompatible
)
