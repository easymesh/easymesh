call :build_all linux 386
call :build_all linux amd64

call :build_all linux arm
call :build_all linux arm64

call :build_all windows 386 .exe
call :build_all windows amd64 .exe


exit /b 0

:build_all
    set GOOS=%1
    set GOARCH=%2
    set TAG=%3

    echo build %GOOS% %GOARCH%

    rmdir /q/s easymesh
    mkdir easymesh

    go build -ldflags="-w -s" -o easymesh\transfer%TAG% transfer\main.go
    go build -ldflags="-w -s" -o easymesh\gateway%TAG% gateway\main.go

    tar -zcf easymesh_%GOOS%_%GOARCH%.tar.gz easymesh
	rmdir /q/s easymesh
	
goto :eof

