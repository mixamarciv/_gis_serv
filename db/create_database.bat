@SET PATH=%PATH%;c:\Program Files (x86)\Firebird\Firebird_2_5\
@SET PATH=%PATH%;c:\Program Files (x86)\Firebird\Firebird_2_5\bin
@SET PATH=%PATH%;c:\Program Files\Firebird\Firebird_2_5\
@SET PATH=%PATH%;c:\Program Files\Firebird\Firebird_2_5\bin
@SET PATH=%PATH%;c:\Program Files (x86)\Firebird\Firebird_2_1\
@SET PATH=%PATH%;c:\Program Files (x86)\Firebird\Firebird_2_1\bin
@SET PATH=%PATH%;c:\Program Files\Firebird\Firebird_2_1\
@SET PATH=%PATH%;c:\Program Files\Firebird\Firebird_2_1\bin

@SET db_file_path=%~dp0
@SET path_prefix=temp_path_for_create_db

@SET dbfilename=db1.fdb

@cd %db_file_path%

@mkdir "%db_file_path%\%path_prefix%\temp\database"


@echo CREATE DATABASE '%db_file_path%\%dbfilename%' page_size 8192 user 'SYSDBA' password 'masterkey'; > "%db_file_path%\%path_prefix%\temp\database\0001_create.sql"
@isql -i "%db_file_path%\%path_prefix%\temp\database\0001_create.sql"

@echo CONNECT '%db_file_path%\%dbfilename%' user 'SYSDBA' password 'masterkey'; > "%db_file_path%\%path_prefix%\temp\database\0002_connect.sql"

@cd create_db_scripts
@mkdir "%db_file_path%\%path_prefix%\temp\database\create_db_scripts"
@for /r . %%g in (*.sql) do (
  @copy "%db_file_path%\%path_prefix%\temp\database\0002_connect.sql"+"%%g" "%db_file_path%\%path_prefix%\temp\database\create_db_scripts\%%~ng"
  @isql -i "%db_file_path%\%path_prefix%\temp\database\create_db_scripts\%%~ng"
)

::!!!!!!!!!!!!!!!!!!!!!!  AAAATTTENTION !!!!!!!!!!!!!!!!!!!!!!!!
@rmdir "%db_file_path%\%path_prefix%" /q /s
::!!!!!!!!!!!!!!!!!!!!!! /AAAATTTENTION !!!!!!!!!!!!!!!!!!!!!!!!

@pause
