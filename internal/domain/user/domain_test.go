package user

import "testing"

func TestInitDomainInitializesSingleton(t *testing.T) {
	domain := InitDomain(nil)
	t.Cleanup(func() {
		currentDomain = nil
	})

	if getDomain() != domain {
		t.Fatal("getDomain()未返回InitDomain()初始化的实例")
	}
	if getRepo() == nil {
		t.Fatal("getRepo()返回nil")
	}

	defer func() {
		if recover() == nil {
			t.Fatal("重复调用InitDomain()没有触发panic")
		}
	}()
	InitDomain(nil)
}
