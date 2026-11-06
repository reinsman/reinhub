package self

import "testing"

func TestInitDomainInitializesSingleton(t *testing.T) {
	domain := InitDomain(&fakeUserNumberGetter{})
	t.Cleanup(func() {
		currentDomain = nil
	})

	if getDomain() != domain {
		t.Fatal("getDomain()未返回InitDomain()初始化的实例")
	}

	defer func() {
		if recover() == nil {
			t.Fatal("重复调用InitDomain()没有触发panic")
		}
	}()
	InitDomain(&fakeUserNumberGetter{})
}
