import 'package:flutter_bloc/flutter_bloc.dart';
import '../models/sftp_file.dart';
import '../services/sftp_service.dart';

// Events
abstract class SftpEvent {}

class SftpListDirectory extends SftpEvent {
  final String path;
  SftpListDirectory(this.path);
}

class SftpUploadFile extends SftpEvent {
  final String localPath;
  final String remotePath;
  SftpUploadFile(this.localPath, this.remotePath);
}

class SftpDownloadFile extends SftpEvent {
  final String remotePath;
  final String localPath;
  SftpDownloadFile(this.remotePath, this.localPath);
}

class SftpDeleteFile extends SftpEvent {
  final String remotePath;
  SftpDeleteFile(this.remotePath);
}

class SftpRenameFile extends SftpEvent {
  final String oldPath;
  final String newPath;
  SftpRenameFile(this.oldPath, this.newPath);
}

class SftpCreateDirectory extends SftpEvent {
  final String remotePath;
  SftpCreateDirectory(this.remotePath);
}

// States
abstract class SftpState {}

class SftpInitial extends SftpState {}

class SftpLoading extends SftpState {}

class SftpDirectoryLoaded extends SftpState {
  final String path;
  final List<SftpFile> files;
  SftpDirectoryLoaded(this.path, this.files);
}

class SftpError extends SftpState {
  final String message;
  SftpError(this.message);
}

class SftpActionSuccess extends SftpState {
  final String message;
  SftpActionSuccess(this.message);
}

// Bloc
class SftpBloc extends Bloc<SftpEvent, SftpState> {
  final SftpService _service;

  SftpBloc({required SftpService service})
      : _service = service,
        super(SftpInitial()) {
    on<SftpListDirectory>(_onListDirectory);
    on<SftpUploadFile>(_onUploadFile);
    on<SftpDownloadFile>(_onDownloadFile);
    on<SftpDeleteFile>(_onDeleteFile);
    on<SftpRenameFile>(_onRenameFile);
    on<SftpCreateDirectory>(_onCreateDirectory);
  }

  Future<void> _onListDirectory(SftpListDirectory event, Emitter<SftpState> emit) async {
    emit(SftpLoading());
    try {
      final files = await _service.listDirectory(event.path);
      emit(SftpDirectoryLoaded(event.path, files));
    } catch (e) {
      emit(SftpError(e.toString()));
    }
  }

  Future<void> _onUploadFile(SftpUploadFile event, Emitter<SftpState> emit) async {
    try {
      await _service.upload(event.localPath, event.remotePath);
      emit(SftpActionSuccess('File uploaded'));
    } catch (e) {
      emit(SftpError(e.toString()));
    }
  }

  Future<void> _onDownloadFile(SftpDownloadFile event, Emitter<SftpState> emit) async {
    try {
      await _service.download(event.remotePath, event.localPath);
      emit(SftpActionSuccess('File downloaded'));
    } catch (e) {
      emit(SftpError(e.toString()));
    }
  }

  Future<void> _onDeleteFile(SftpDeleteFile event, Emitter<SftpState> emit) async {
    try {
      await _service.delete(event.remotePath);
      emit(SftpActionSuccess('File deleted'));
    } catch (e) {
      emit(SftpError(e.toString()));
    }
  }

  Future<void> _onRenameFile(SftpRenameFile event, Emitter<SftpState> emit) async {
    try {
      await _service.rename(event.oldPath, event.newPath);
      emit(SftpActionSuccess('File renamed'));
    } catch (e) {
      emit(SftpError(e.toString()));
    }
  }

  Future<void> _onCreateDirectory(SftpCreateDirectory event, Emitter<SftpState> emit) async {
    try {
      await _service.createDirectory(event.remotePath);
      emit(SftpActionSuccess('Directory created'));
    } catch (e) {
      emit(SftpError(e.toString()));
    }
  }
}
