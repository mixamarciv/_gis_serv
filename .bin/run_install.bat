::получаем curpath:
@FOR /f %%i IN ("%0") DO SET curpath=%~dp0
::задаем основные переменные окружения
@CALL "%curpath%/set_path.bat"


@del app.exe
@CLS

@echo === install ===================================================================
go get -u "github.com/gorilla/mux"
go get -u "github.com/mixamarciv/gofncstd3000"
go get -u "github.com/parnurzeal/gorequest"
go get -u github.com/go-ini/ini

go get -u github.com/cznic/mathutil
go get -u github.com/nyarla/go-crypt
go get -u github.com/nakagami/firebirdsql


::go get -u "github.com/satori/go.uuid"
::go get -u "github.com/parnurzeal/gorequest"
::go get -u "github.com/palantir/stacktrace"
::go get -u "github.com/gosuri/uilive"
::"github.com/mixamarciv/gofncstd3000"

::библиотека для работы с XMLками
go get -u "github.com/jteeuwen/go-pkg-xmlx"

go install



@echo ==== end ======================================================================
@PAUSE
