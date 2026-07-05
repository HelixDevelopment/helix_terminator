// TODO: implement auth service with secure token storage

class AuthService {
  Future<void> login(String email, String password) async {
    // TODO: call auth API and store tokens
  }

  Future<void> logout() async {
    // TODO: clear tokens
  }

  Future<bool> isAuthenticated() async {
    // TODO: check token validity
    return false;
  }
}
