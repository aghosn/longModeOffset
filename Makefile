all: main

main: main.go kvm.go kvm_consts.go
	gosb build $^

.PHONY: clean

clean: 
	rm main
